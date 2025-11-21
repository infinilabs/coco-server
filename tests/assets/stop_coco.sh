pkill coco

while nc -z 127.0.0.1 9000; do
    echo "Coco Server is still running. Waiting for it to exit..."
    sleep 5
done
while nc -z 127.0.0.1 2900; do
    echo "Coco Server is still running. Waiting for it to exit..."
    sleep 5
done