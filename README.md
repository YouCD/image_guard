# image_guard
用于监视容器镜像是否更新的工具类似`watchtower`，但是`watchtower`不能从`registry-mirrors`地址获取？


# docker
```shell
git clone github.com/YouCD/image_guard
cd image_guard
docker build  --network=host  -t image_guard  .
docker run --rm --name image_guard -v /var/run/docker.sock:/var/run/docker.sock image_guard 
```