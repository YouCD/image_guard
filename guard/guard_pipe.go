package guard

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/youcd/toolkit/log"
	"strings"
	"sync"
)

var (
	guard *Guard
	once  sync.Once
)

func Pipe(containers []string) {
	ctx := context.Background()
	var err error
	once.Do(func() {
		guard, err = NewGuard(ctx, containers)
		if err != nil {
			panic(err)
		}
	})

	containerList := getContainerJSONByName(ctx, guard.containers...)
	if len(containerList) == 0 {
		log.Error("no containers found")
		return
	}

	var containerInfos []*containerInfo
	for _, c := range containerList {
		createTime, repoTag, err := guard.ImageCreateTimeAndRepoTag(ctx, c)
		if err != nil {
			log.Warnf("get image create time error: %v", err)
			continue
		}
		containerInfos = append(containerInfos, &containerInfo{
			Name: strings.TrimPrefix(c.Name, "/"),
			CurrentImage: &imageInfo{
				Repository:  repoTag,
				ImageCreate: createTime,
				ImageID:     c.Image,
			},
		})
	}

	mirrors, err := guard.ListRegistryMirrors(ctx)
	if err != nil {
		log.Errorf("get registry mirrors error: %v", err)
		return
	}
	for _, info := range containerInfos {
		info.RegistryMirrors = append(info.RegistryMirrors, mirrors...)
	}

	for _, info := range containerInfos {
		if err := guard.RegistryMirrorInspect(info); err != nil {
			log.Debugf("registry inspect error: %v", err)
		}
	}

	updateMap := make(map[string]*imageInfo)
	for _, info := range containerInfos {
		for _, img := range info.RemoteImages {
			if info.CurrentImage.ImageID == img.ImageID {
				log.Infow("checkImageID", "container", info.Name, "ImageID", img.ImageID)
				break
			}
			if img.ImageCreate.After(info.CurrentImage.ImageCreate) {
				if curr, exists := updateMap[info.Name]; !exists || img.ImageCreate.After(curr.ImageCreate) {
					updateMap[info.Name] = img
					log.Debugf("found newer image: %s -> %s", info.Name, img.Repository)
				}
				break
			}
		}
	}

	var wg sync.WaitGroup
	for name, img := range updateMap {
		wg.Add(1)
		go func(name string, img *imageInfo) {
			defer wg.Done()

			newImage := fmt.Sprintf("%s:%s", img.Reference.Context().RepositoryStr(), img.Reference.Identifier())
			log.Infof("updating container: %s to release time:  %s", name, img.ImageCreate.Format("2006-01-02 15:04:05"))
			if err := guard.ImagePull(ctx, img.Repository, func(_ string) string { return newImage }); err != nil {
				log.Errorf("pull image error: %v", err)
				return
			}
			if err := guard.UpdateContainerImage(ctx, name, newImage); err != nil {
				log.Errorf("update container error: %v", err)
			}
		}(name, img)
	}
	wg.Wait()

	// todo: clean up unused images
}

func getContainerJSONByName(ctx context.Context, names ...string) []types.ContainerJSON {
	var list []types.ContainerJSON
	for _, name := range names {
		inspect, err := guard.DockerCLIClient.Client().ContainerInspect(ctx, name)
		if err != nil {
			log.Error(err)
			continue
		}
		list = append(list, inspect)
	}
	return list
}
