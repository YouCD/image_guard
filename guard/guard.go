package guard

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/youcd/toolkit/docker"
	"github.com/youcd/toolkit/log"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type imageInfo struct {
	ImageCreate time.Time
	Repository  string
	Reference   name.Reference
	ImageID     string
}
type containerInfo struct {
	Name            string
	CurrentImage    *imageInfo
	RemoteImages    []*imageInfo
	RegistryMirrors []string
}
type Guard struct {
	*docker.Docker
	containers []string
}

func NewGuard(ctx context.Context, containers []string) (*Guard, error) {
	newDocker, err := docker.NewDocker(ctx)
	if err != nil {
		return nil, err
	}

	list := containers
	if len(containers) == 0 {
		containerList, err := newDocker.ContainerList(ctx)
		if err != nil {
			log.Errorf("get container list error: %v", err)
			return nil, err
		}
		for _, c := range containerList {
			list = append(list, strings.TrimPrefix(c.Names[0], "/"))
		}
	}
	log.Infof("watch container: %s", strings.Join(list, ","))
	return &Guard{Docker: newDocker, containers: list}, nil
}
func (g *Guard) UpdateContainerImage(ctx context.Context, containerName, image string) error {
	// 获取容器配置
	inspectJSON, err := g.Inspect(ctx, containerName)
	if err != nil {
		return err
	}
	// 删除旧容器
	if err = g.ContainerRemove(ctx, containerName); err != nil {
		return err
	}

	// 更新镜像字段
	inspectJSON.Config.Image = image

	// 构建新的网络配置：只保留网络名，不复制 IP/MAC
	EndpointsConfig := make(map[string]*network.EndpointSettings)
	for netName := range inspectJSON.NetworkSettings.Networks {
		EndpointsConfig[netName] = &network.EndpointSettings{}
	}
	networkingConfig := &network.NetworkingConfig{EndpointsConfig: EndpointsConfig}

	// 创建并启动容器
	for {
		resp, err := g.ContainerCreate(ctx, containerName, inspectJSON, networkingConfig)
		if err != nil {
			return err
		}

		if err := g.DockerCLIClient.Client().ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
			// 某些容器无用户配置
			if strings.Contains(err.Error(), "no matching entries in passwd file") {
				_ = g.ContainerRemove(ctx, containerName)
				inspectJSON.Config.User = ""
				continue
			}
			return fmt.Errorf("ContainerStart() error: %w", err)
		}
		break
	}
	return nil
}
func (g *Guard) ImageCreateTimeAndRepoTag(ctx context.Context, containerJSON types.ContainerJSON) (time.Time, string, error) {
	imageInspect, _, err := g.DockerCLIClient.Client().ImageInspectWithRaw(ctx, containerJSON.Image)
	if err != nil {
		return time.Time{}, "", err
	}
	parse, err := time.Parse(time.RFC3339, imageInspect.Created)
	if err != nil {
		return time.Time{}, "", err
	}
	log.Debugf("ImageCreateTimeAndRepoTag: %s, %s", parse, imageInspect.RepoTags[0])
	return parse, imageInspect.RepoTags[0], nil
}

func (g *Guard) ListRegistryMirrors(ctx context.Context, ) ([]string, error) {
	info, err := g.DockerCLIClient.Client().Info(ctx)
	if err != nil {
		return nil, err
	}
	return info.RegistryConfig.Mirrors, err
}

func (g *Guard) RegistryInspect(container *containerInfo) error {
	var refs []name.Reference
	var errs []error
	for _, mirror := range container.RegistryMirrors {
		ref, err := newRef(mirror, container.CurrentImage.Repository)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		log.Debugf("RegistryMirror: %s  Repository: %s", ref, container.CurrentImage.Repository)
		refs = append(refs, ref)
	}

	for _, ref := range refs {
		// 3. 获取镜像的配置信息
		img, err := getRemoteImage(ref)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		file, _ := img.ConfigFile()
		//if err != nil {
		//	errs = append(errs, err)
		//	continue
		//}

		// ConfigName() -> Config 的哈希（即 "IMAGE ID"）
		configHash, _ := img.ConfigName()

		//digest, err := img.Digest()
		//if err != nil {
		//	errs = append(errs, err)
		//	continue
		//}

		container.RemoteImages = append(container.RemoteImages, &imageInfo{
			ImageCreate: file.Created.Time,
			Repository:  ref.String(),
			Reference:   ref,
			ImageID:     configHash.String(),
		})
	}

	if len(errs) > 0 && len(container.RemoteImages) == 0 {
		return errors.Join(errs...)
	}
	return nil
}

func newRef(registryMirrors string, repoTag string, ) (name.Reference, error) {
	if len(registryMirrors) != 0 {
		parse, _ := url.Parse(registryMirrors)
		repoTag = fmt.Sprintf("%s/%s", parse.Host, repoTag)
	}

	// 2. 解析镜像引用
	return name.ParseReference(repoTag)
}

// getRemoteImage 获取镜像的配置信息
func getRemoteImage(ref name.Reference) (v1.Image, error) {
	// 获取远程镜像
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return nil, fmt.Errorf("获取远程镜像失败: %v", err)
	}
	// 获取镜像配置文件
	return img, nil
}
