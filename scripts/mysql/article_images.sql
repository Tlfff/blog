USE blog;

CREATE TABLE IF NOT EXISTS article_images(
    id                 BIGINT UNSIGNED          NOT NULL    AUTO_INCREMENT              COMMENT '图片ID',
    article_id         BIGINT UNSIGNED                                              COMMENT '所属文章ID，未绑定时为空',
    object_key         VARCHAR(255)             NOT NULL                                COMMENT '对象存储Key',
    created_time       DATETIME                 NOT NULL    DEFAULT CURRENT_TIMESTAMP   COMMENT '创建时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_object_key (object_key),
    KEY idx_article_id (article_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = '文章图片表'
