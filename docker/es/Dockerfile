FROM docker.elastic.co/elasticsearch/elasticsearch:8.19.20

# 安装中文分词插件
RUN bin/elasticsearch-plugin install --batch \
    https://get.infini.cloud/elasticsearch/analysis-ik/8.19.20

# 安装拼音分词插件
RUN bin/elasticsearch-plugin install --batch \
    https://get.infini.cloud/elasticsearch/analysis-pinyin/8.19.20
