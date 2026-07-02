CREATE TABLE article_view_histories(
    id                 BIGINT UNSIGNED          NOT NULL    AUTO_INCREMENT              COMMENT '浏览历史ID',
    user_id            BIGINT UNSIGNED          NOT NULL                                COMMENT '用户ID',
    article_id         BIGINT UNSIGNED          NOT NULL                                COMMENT '文章ID',
    created_time       DATETIME                 NOT NULL    DEFAULT CURRENT_TIMESTAMP   COMMENT '创建时间',
    updated_time       DATETIME                 NOT NULL    DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '修改时间',
    PRIMARY KEY (id),
    KEY idx_userid_createdtime (user_id,created_time),
    -- KEY idx_updated_time (updated_time)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '浏览历史表'
