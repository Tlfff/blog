# 文章搜索运行与运维说明

## 1. 运行边界

文章搜索属于模块化单体中的 Search 上下文，不是独立微服务。三个入口分别承担：

```text
blog server            提供 GET /article/search
blog search-sync       消费 Canal 批次并增量维护 Elasticsearch
blog search-rebuild    从 MySQL 全量创建新物理索引并切换别名
```

MySQL Article 数据是唯一真相，Elasticsearch 是可重建投影。文章创建、编辑、发布和删除不依赖 Elasticsearch，可接受数秒级搜索同步延迟。

## 2. 首次上线

首次上线必须按以下顺序执行：

1. 部署启用 ROW/FULL binlog 的 MySQL、带 IK/拼音插件的 Elasticsearch 和 Canal。
2. 保持 `search-sync` 停止，避免全量构建期间继续写旧别名。
3. 执行 `blog search-rebuild`，等待新版本索引创建、批量导入、数量校验和 `article_search` 别名切换完成。
4. 启动单实例 `blog search-sync`。
5. 检查 Canal 积压事件被顺序回放，并验证文章发布、编辑、转草稿和删除对应的 upsert/delete 行为。
6. 开放或验证 `GET /article/search`。

## 3. 日常重建

需要修改 mapping、分析器或修复索引数据时：

1. 停止 `search-sync`。
2. 执行 `search-rebuild`。
3. 验证新物理索引数量和典型中文、完整拼音、正文、标签查询。
4. 恢复 `search-sync`。
5. 观察 Canal 位点推进并确认停机期间的积压变更已经回放。

重建使用版本化物理索引和原子 alias 切换，不直接清空当前在线索引。成功切换后保留上一物理索引，旧索引只能通过显式运维操作删除。

## 4. 回滚和故障排查

- 新索引构建失败：别名保持不变，命令会尝试删除失败的新索引。
- 别名切换后结果异常：先停止 `search-sync`，再把 `article_search` 别名切回上一物理索引。
- Canal 批次持续阻塞：检查日志中的 batch、文章字段转换和 Elasticsearch Bulk 条目错误；未成功批次不会 ack。
- 同步延迟增长：检查 Canal 连接、MySQL binlog 保留窗口和 Elasticsearch 写入耗时。
- 拼音查询失败：确认 Elasticsearch 版本与 IK、拼音插件版本完全一致，并使用 analyzer API 检查完整拼音 token。
- Elasticsearch 不可用：搜索接口返回搜索服务暂不可用，不会降级为 MySQL `LIKE`；Article 写接口不受影响。

生产环境必须限制 Elasticsearch 和 Canal 的网络访问范围，Elasticsearch 凭证通过 Secret 注入，禁止直接使用开发环境的无认证公开端口配置。
