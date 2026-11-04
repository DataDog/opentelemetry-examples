const { Kafka, logLevel } = require('kafkajs')

const broker = process.env.KAFKA_ADDR || 'kafka:9092'
const inputTopic = process.env.KAFKA_TOPIC || 'java-forwarder-orders'
const outputTopic = process.env.KAFKA_OUTPUT_TOPIC || 'node-forwarder-orders'

const log = (msg, fields = {}) => console.log(JSON.stringify({ level: 'info', msg, ...fields }))

const kafka = new Kafka({ clientId: 'node-forwarder', brokers: [broker], logLevel: logLevel.WARN })
const admin = kafka.admin()
const consumer = kafka.consumer({ groupId: 'node-forwarder' })
const producer = kafka.producer()

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

function processingDelayMs () {
  const gaussian = Math.sqrt(-2 * Math.log(1 - Math.random())) * Math.cos(2 * Math.PI * Math.random())
  return Math.min(Math.max(500 + 150 * gaussian, 0), 990)
}

async function main () {
  await admin.connect()
  await admin.createTopics({ topics: [{ topic: inputTopic }] })
  await admin.disconnect()
  await Promise.all([consumer.connect(), producer.connect()])
  await consumer.subscribe({ topic: inputTopic, fromBeginning: true })
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
