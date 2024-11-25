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

__easysearch() {
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

    coco_exe=`find ~+  -maxdepth 1 -perm -111 -type f -name "*coco*"`

    echo $espw | $coco_exe keystore add --force --stdin  ES_PASSWORD
}

__main() {
    __ollama
    __ocr_server
    __easysearch

    if [ $? -eq 0 ]; then
        echo "all set"
    else
        echo "failed to init, you may try manually go step by step"
    fi
}

__main $@
