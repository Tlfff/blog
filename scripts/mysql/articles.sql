CREATE TABLE articles(
    id                 BIGINT UNSIGNED          NOT NULL    AUTO_INCREMENT              COMMENT '文章ID',
    author_id          BIGINT UNSIGNED          NOT NULL                                COMMENT '作者ID',
    title              VARCHAR(255)             NOT NULL                                COMMENT '文章标题',
    content            TEXT                     NOT NULL                                COMMENT '文章正文(支持Markdown)',
    tags               VARCHAR(255)             NOT NULL                                COMMENT '文章标签',
    view_count         INT UNSIGNED             NOT NULL    DEFAULT 0                   COMMENT '浏览量',
    like_count         INT UNSIGNED             NOT NULL    DEFAULT 0                   COMMENT '点赞数',
    comment_count      INT UNSIGNED             NOT NULL    DEFAULT 0                   COMMENT '评论数',
    status             TINYINT                  NOT NULL    DEFAULT 1                   COMMENT '文章状态:0-已删除,1-草稿,2-已发表',
    created_time       DATETIME                 NOT NULL    DEFAULT CURRENT_TIMESTAMP   COMMENT '创建时间',
    updated_time       DATETIME                 NOT NULL    DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    PRIMARY KEY (id),
    KEY idx_created_time (created_time),
    KEY idx_updated_time (updated_time)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '文章表'
