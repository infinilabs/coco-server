nohup ./bin/coco &

while ! nc -z 127.0.0.1 9000; do
    echo "Coco Server is not up. Will re-check in 5 seconds..."
    sleep 5
done
while ! nc -z 127.0.0.1 2900; do
    echo "Coco Server is not up. Will re-check in 5 seconds..."
    sleep 5
done