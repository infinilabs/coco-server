#!/bin/bash 

pwd=`pwd`

__ollama() {
    which ollama || curl -fsSL https://ollama.com/install.sh | sh
    OLLAMA_HOST=0.0.0.0:11434 ollama serve
    ollama pull llama3.2
    ollama pull llama2-chinese:13b
}

__ocr_server() {
    docker run --name ocrserver \
        -p 8080:8080 \
        -d \
        otiai10/ocrserver
}

__easyserch() {
    [ -d $pwd/easysearch ] || mkdir -p $pwd/easysearch/{data,logs}
    sudo chown -R 602:602 $pwd/easysearch

    docker run \
        -itd --name easysearch \
        --hostname easysearch \
        --restart=unless-stopped \
        -p 9200:9200 \
        -v $pwd/easysearch/data:/app/easysearch/data:rw \
        -v $pwd/easysearch/logs:/app/easysearch/logs:rw \
        infinilabs/easysearch:1.8.3-265

    espw=$(docker logs easysearch | grep "admin:" | head -n 1 | cut -d ':' -f 2 | cut -d ' ' -f 1)
    echo Easysearch admin password: $espw

    [ -f coco.yml ] && cp coco.yml coco.yml.bak

    egrep -lZ "\%ES_PASSWD\%" ${pwd}/coco.yml \
      | xargs -0 -l sed -i -e "s/\%ES_PASSWD\%/${espw}/g"
}

__main() {
    __ollama
    __ocr_server
    __easyserch

    if [ $? -eq 0 ]; then
        echo "all set"
    else
        echo "failed to init, you may try manually go step by step"
    fi
}

__main $@
