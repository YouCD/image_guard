package guard

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/youcd/toolkit/log"
	"strings"
	"sync"
	"time"
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

	listContainerJSON := getContainerJSONByName(ctx, guard.containers...)
	if len(listContainerJSON) == 0 {
		log.Errorf("no containerJSON found")
		return
	}

	//	1. 获取 当前容器对应 镜的 的 Created 时间
	var containerInfoList []*containerInfo
	for _, containerJSON := range listContainerJSON {
		createTime, repoTag, err := guard.ImageCreateTimeAndRepoTag(ctx, containerJSON)
		if err != nil {
			log.Warn("get image create time error: %v", err)
			continue
		}
		containerInfoList = append(containerInfoList, &containerInfo{
			Name: strings.TrimPrefix(containerJSON.Name, "/"),
			CurrentImage: &imageInfo{
				Repository:  repoTag,
				ImageCreate: createTime,
				ImageID:     containerJSON.Image,
			},
		})
	}

	//  2. 获取 registry-mirrors
	mirrors, err := guard.ListRegistryMirrors(ctx)
	if err != nil {
		log.Errorf("get registry mirrors error: %v", err)
		return
	}
	for _, mirror := range mirrors {
		log.Debugf("registry mirror: %s", mirror)
		for _, info := range containerInfoList {
			info.RegistryMirrors = append(info.RegistryMirrors, mirror)
		}
	}

	//  3. 获取 registry 中对应镜像的 Created 时间
	for _, info := range containerInfoList {

		if err := guard.RegistryInspect(info); err != nil {
			log.Errorf("get registry inspect error: %v", err)
			continue
		}
	}
	updateImageMap := map[string]*imageInfo{}
	for _, info := range containerInfoList {
		var check bool
		for _, image := range info.RemoteImages {

			if info.CurrentImage.ImageID == image.ImageID {
				log.Infof("No need to update the image, container: %s", info.Name)
				break
			}
			// 判断镜像是否比当前镜像更新
			if image.ImageCreate.After(info.CurrentImage.ImageCreate) {
				if c, ok := updateImageMap[info.Name]; ok {
					// 如果已有记录，再比较时间，保留时间更新的
					if image.ImageCreate.After(c.ImageCreate) {
						updateImageMap[info.Name] = image
						log.Debugf("add update container: %s, ref: %s, create: %s", info.Name, image.Repository, image.ImageCreate.Format(time.DateTime))
					}
				} else {
					// 没有记录，直接放入 map
					updateImageMap[info.Name] = image
					log.Debugf("add update container: %s, ref: %s, create: %s", info.Name, image.Repository, image.ImageCreate.Format(time.DateTime))
				}

				break // 跳出 RemoteImages 循环
			} else {
				if !check {
					log.Infof("No need to update the image, container: %s", info.Name)
				} else {
					continue
				}
				check = true
			}
		}
	}

	for name, image := range updateImageMap {
		newImage := image.Repository
		newTagFunc := func(string) string {
			log.Infof("update container:%s,ref:%s", name, image.Repository)
			newImage = fmt.Sprintf("%s:%s", image.Reference.Context().RepositoryStr(), image.Reference.Identifier())
			return newImage
		}
		//  5. 如果更新 就拉取最新的镜像
		if err = guard.ImagePull(ctx, image.Repository, newTagFunc); err != nil {
			log.Errorf("pull image error: %v", err)
			continue
		}
		log.Infof("update container:%s -> %s, created: %s", name, newImage, image.ImageCreate.Format(time.DateTime))
		//  6. 重新创建新的容器
		if err := guard.UpdateContainerImage(ctx, name, newImage); err != nil {
			log.Errorf("update image error: %v", err)
			continue
		}
	}
	// 7. 其他操作
	//		清理多余的镜像
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

func getFromArgs(ctx context.Context, containers ...string) []types.ContainerJSON {
	var list []types.ContainerJSON
	for _, s := range containers {
		inspect, err := guard.DockerCLIClient.Client().ContainerInspect(ctx, s)
		if err != nil {
			log.Error(err)
			continue
		}
		list = append(list, inspect)
	}
	return list
}
