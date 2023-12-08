## Distributed Words Counting System

A distributed application designed to calculate the frequency of user-inputted words

* User sends words to `/words`
* Severs splits the message and sends each word as a Kafka message
* Consumer batches the messages and processes them after receiving 5 messages

![image](https://github.com/DataDog/opentelemetry-examples/assets/2471669/c7e2fde7-e49f-4678-94ae-a72b9db0ccbf)

Request
```
curl -X POST -d 'word,word,hello,hello,world,world' http://localhost:9090/words
```
Response
```{}```

## Docker Compose

Retrieve API_KEY from datadoghq, and expose same on Shell

```
export DD_API_KEY=xx
```
Bring up the services

```
docker compose -f docker-compose-otel.yaml  up
```

