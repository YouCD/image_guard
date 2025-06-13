package guard

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/youcd/toolkit/docker"
	"github.com/youcd/toolkit/log"
)

var ErrGetRemoteImage = errors.New("get remote image error")

type (
	imageInfo struct {
		ImageCreate time.Time
		Repository  string
		Reference   name.Reference
		ImageID     string
	}
	containerInfo struct {
		Name            string
		CurrentImage    *imageInfo
		RemoteImages    []*imageInfo
		RegistryMirrors []string
	}
	Guard struct {
		*docker.Docker
		containers []string
	}
)

var (
	defaultTransport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   50,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
)

func NewGuard(ctx context.Context, containers []string) (*Guard, error) {
	d, err := docker.NewDocker(ctx)
	if err != nil {
		return nil, err
	}

	if len(containers) == 0 {
		cl, err := d.ContainerList(ctx)
		if err != nil {
			log.Errorf("get container list error: %v", err)
			return nil, err
		}
		for _, c := range cl {
			containers = append(containers, strings.TrimPrefix(c.Names[0], "/"))
		}
	}
	log.Infof("watch containers: %s", strings.Join(containers, ","))
	return &Guard{Docker: d, containers: containers}, nil
}

func (g *Guard) UpdateContainerImage(ctx context.Context, name, image string) error {
	inspect, err := g.Inspect(ctx, name)
	if err != nil {
		return err
	}
	if err := g.ContainerRemove(ctx, name); err != nil {
		return err
	}
	inspect.Config.Image = image

	nets := make(map[string]*network.EndpointSettings)
	for n := range inspect.NetworkSettings.Networks {
		nets[n] = &network.EndpointSettings{}
	}
	networking := &network.NetworkingConfig{EndpointsConfig: nets}

	for {
		resp, err := g.ContainerCreate(ctx, name, inspect, networking)
		if err != nil {
			return err
		}
		if err := g.DockerCLIClient.Client().ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
			if strings.Contains(err.Error(), "no matching entries in passwd file") {
				_ = g.ContainerRemove(ctx, name)
				inspect.Config.User = ""
				continue
			}
			return fmt.Errorf("ContainerStart() error: %w", err)
		}
		break
	}
	return nil
}

func (g *Guard) ImageCreateTimeAndRepoTag(ctx context.Context, container types.ContainerJSON) (time.Time, string, error) {
	img, _, err := g.DockerCLIClient.Client().ImageInspectWithRaw(ctx, container.Image)
	if err != nil {
		return time.Time{}, "", err
	}
	created, err := time.Parse(time.RFC3339, img.Created)
	if err != nil {
		return time.Time{}, "", err
	}
	if len(img.RepoTags) == 0 {
		return created, "", nil
	}
	log.Debugf("ImageCreateTimeAndRepoTag: %s, %s", created, img.RepoTags[0])
	return created, img.RepoTags[0], nil
}

func (g *Guard) ListRegistryMirrors(ctx context.Context) ([]string, error) {
	info, err := g.DockerCLIClient.Client().Info(ctx)
	if err != nil {
		return nil, err
	}
	return info.RegistryConfig.Mirrors, nil
}

func (g *Guard) RegistryMirrorInspect(c *containerInfo) error {
	refMap := make(map[string]struct{})
	var refs []name.Reference
	var errs []error

	for _, mirror := range c.RegistryMirrors {
		log.Infow("RegistryMirrorInspect", "container", c.Name, "registryMirror", strings.TrimSuffix(mirror, "/"))
		ref, err := newRef(mirror, c.CurrentImage.Repository)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if _, ok := refMap[ref.Name()]; !ok {
			refMap[ref.Name()] = struct{}{}
			refs = append(refs, ref)
		}
	}

	var remoteErr error
	for _, ref := range refs {
		img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain), remote.WithTransport(defaultTransport))
		if err != nil {
			if errors.Is(err, ErrGetRemoteImage) {
				remoteErr = ErrGetRemoteImage
				continue
			}
			log.Errorw("getRemoteImage", "err", err)
			errs = append(errs, err)
			continue
		}
		file, err := img.ConfigFile()
		if err != nil {
			log.Error(err)
			errs = append(errs, err)
			continue
		}
		id, _ := img.ConfigName()
		c.RemoteImages = append(c.RemoteImages, &imageInfo{
			ImageCreate: file.Created.Time,
			Repository:  ref.String(),
			Reference:   ref,
			ImageID:     id.String(),
		})
		log.Infow("RegistryMirrorInspect", "container", c.Name, "ref", ref.Context(), "image", ref.String())
	}
	if remoteErr != nil {
		errs = append(errs, remoteErr)
	}
	if len(errs) > 0 && len(c.RemoteImages) == 0 {
		return errors.Join(errs...)
	}
	return nil
}

func newRef(mirror, repoTag string) (name.Reference, error) {
	ref, err := name.ParseReference(repoTag)
	if err != nil {
		return nil, err
	}
	if ref.Context().RegistryStr() == "index.docker.io" && mirror != "" {
		u, err := url.Parse(mirror)
		if err != nil {
			return nil, err
		}
		if u.Scheme == "http" {
			return name.NewTag(fmt.Sprintf("%s/%s", u.Host, repoTag), name.Insecure)
		}
		return name.NewTag(fmt.Sprintf("%s/%s", u.Host, repoTag))
	}
	return ref, nil
}
