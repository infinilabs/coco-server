#!/bin/bash 

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
    docker run \
        -itd --name easysearch \
        -p 9200:9200 \
        infinilabs/easysearch:1.8.3-265

    pw=$(docker logs easysearch | grep "admin:" | head -n 1 | cut -d ':' -f 2 | cut -d ' ' -f 1)
    echo Easysearch admin password: $pw
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
