curl -X DELETE 'http://127.0.0.1:9200/coco_*'
echo # to add a newline char
curl -X POST 'http://127.0.0.1:9200/_snapshot/repo_ezs/coco_indices/_restore?wait_for_completion=true'
echo # to add a newline char