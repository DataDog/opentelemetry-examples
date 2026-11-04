const { Kafka, logLevel } = require('kafkajs')

const broker = process.env.KAFKA_ADDR || 'kafka:9092'
const inputTopic = process.env.KAFKA_TOPIC || 'java-forwarder-orders'
const outputTopic = process.env.KAFKA_OUTPUT_TOPIC || 'node-forwarder-orders'

const log = (msg, fields = {}) => console.log(JSON.stringify({ level: 'info', msg, ...fields }))

const kafka = new Kafka({ clientId: 'node-forwarder', brokers: [broker], logLevel: logLevel.WARN })
const consumer = kafka.consumer({ groupId: 'node-forwarder' })
const producer = kafka.producer()

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

function processingDelayMs () {
  const gaussian = Math.sqrt(-2 * Math.log(1 - Math.random())) * Math.cos(2 * Math.PI * Math.random())
  return Math.min(Math.max(500 + 150 * gaussian, 0), 990)
}

async function subscribeWhenTopicExists () {
  for (;;) {
    try {
      await consumer.subscribe({ topic: inputTopic, fromBeginning: true })
      return
    } catch (err) {
      if (err.type !== 'UNKNOWN_TOPIC_OR_PARTITION') throw err
      log('waiting for topic to exist', { input_topic: inputTopic })
      await sleep(2000)
    }
  }
}

async function main () {
  await Promise.all([consumer.connect(), producer.connect()])
  await subscribeWhenTopicExists()
  log('node-forwarder started', { input_topic: inputTopic, output_topic: outputTopic })
  await consumer.run({
    eachMessage: async ({ message }) => {
      log('forward_order', { key: message.key?.toString() })
      await sleep(processingDelayMs())
      await producer.send({ topic: outputTopic, messages: [{ key: message.key, value: message.value }] })
    }
  })
}

const shutdown = async () => {
  await Promise.allSettled([consumer.disconnect(), producer.disconnect()])
  process.exit(0)
}
process.on('SIGTERM', shutdown)
process.on('SIGINT', shutdown)

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
