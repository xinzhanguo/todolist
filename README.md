# todolist
a todo list


## 创建数据库表单

```
CREATE TABLE `todos` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `uid` char(128) NOT NULL DEFAULT '' COMMENT 'The uid',
  `content` text,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uid_index` (`uid`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COMMENT='todos table';
```


## 示例

![](https://raw.githubusercontent.com/xinzhanguo/todolist/refs/heads/main/todo1.jpg)

## 使用方法

```
docker build -t todo:1 .
docker run -d --name todo --restart=always todo:1
```