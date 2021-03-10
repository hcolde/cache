# cache



## 安装

```shell
go get github.com/hcolde/cache
```



## 使用

```golang
package main

import (
  "fmt"
  "time"

  "github.com/hcolde/cache"
)

func main() {
  ctx, cancel := context.WithCancel(context.Background())
  defer cancel()

  opt := cache.Options{
    MaxSize:    10,
    RefreshTTL: true,
  }

  c, err := cache.New(ctx, opt)
  if err != nil {
    fmt.Println(err)
    return
  }
  
  v := c.Get("aaa")
  fmt.Println(v) // nil
  
  c.Set("aaa", 1, 1 * time.Second)
  v = c.Get("aaa")
  fmt.Println(v) // 1
  
  time.Sleep(2 * time.Second)
  v = c.Get("aaa")
  fmt.Println(v) // nil
}
```



## 介绍

cache是一个存放在内存中的键值对缓存，键必须是string类型，而值可以是任何类型。它仅限于在单机上使用，而且未来也不会支持分布式，因为[Redis](https://github.com/redis/redis)已经足够好用。

相较于Redis，cache的优点在于：

1. 存取数据时，没有通信带来的延时
2. 值可以是golang中的任何类型
3. 对于仅需要缓存功能的需求场景，更轻量

不建议在以下这些需求场景中使用cache：

1. 在多个节点存取并同步数据
2. 当数据未过期时必须要获取到数据



实现原理：

* **惰性删除**
  * 获取数据时，会检查该数据是否过期，若过期则直接删除，并返回nil，参考自Redis的惰性删除；
  * 使用`ForcedGet`方法获取数据，即使检查到该数据已过期，在删除数据的同时，仍会返回该数据。
* **定时删除**
  * cache会每个100ms检查 $\log_2 cache.size$个数据的ttl，当ttl过期时会将该数据删除；
  * 若此次删除的数据超过cache.size $\times$ 0.25，则继续扫描；
  * 当数据被删除时，**环**的尾部指针指向该数据的index。
* **环**
  * cache使用环`[]string`来存放数据的key和index，初始化时的大小取决于用户自定义；
  * **环**有一个头部指针和一个尾部指针，初始化时头部指针指向0的位置，尾部指针指向cache.size - 1的位置；
  * **环**的作用是方便**定时删除**策略的扫描，并控制缓存的大小。
* **获取数据**
  * 获取数据时触发**惰性删除**策略，当数据被**惰性删除**时，会将数据的index放入`indexPool`中。
* **新增数据**
  * 新增数据时，会优先从`indexPool`中获取index；
  * 当`indexPool`为空时，会将**环**的头部指针指向的index赋予该数据，并向前移动一步，依次保证头部指针总是指向未使用过的index；
  * 当头部指针与尾部指针相遇时，说明**环**的index已经被分配完毕，此时会将尾部指向指向的数据给删除，即时它尚未过期！然后将该index赋予新增的数据。

