'use strict'

// Initialize dd-trace before loading instrumented libraries.
const tracer = require('dd-trace').init()
const provider = new tracer.TracerProvider()
provider.register()

const express = require('express')
const { Pool } = require('pg')
const { trace } = require('@opentelemetry/api')
const { dynamicProbeTarget } = require('./probe-target')

const app = express()
const otelTracer = trace.getTracer('otel-node-validation')
const allocationHold = []
const pool = new Pool({
  host: process.env.POSTGRES_HOST || 'postgres',
  port: Number(process.env.POSTGRES_DB_PORT || 5432),
  database: process.env.POSTGRES_DB || 'validation',
  user: process.env.POSTGRES_USER || 'validation',
  password: process.env.POSTGRES_PASSWORD || 'validation',
  application_name: process.env.POSTGRES_APPLICATION_NAME || process.env.DD_SERVICE,
})

app.get('/health', (_req, res) => res.json({ status: 'ok', time: new Date().toISOString() }))

app.get('/span', async (_req, res) => {
  await new Promise(resolve => setTimeout(resolve, 500))
  otelTracer.startActiveSpan('manual.otel.span', span => {
    span.setAttribute('demo.endpoint', '/span')
    span.setAttribute('validation.bridge', 'dd-trace-node-otel-api')
    span.end()
  })
  res.json({ span: 'ok' })
})

app.get('/probe', (req, res) => {
  const value = Number(req.query.value || 7)
  const result = dynamicProbeTarget(value)
  res.json({ input: value, result })
})

app.get('/cpu', (req, res) => {
  const seconds = Math.min(Math.max(Number(req.query.seconds || 1.5), 0.1), 10)
  const deadline = process.hrtime.bigint() + BigInt(Math.floor(seconds * 1e9))
  let iterations = 0
  let checksum = 0
  while (process.hrtime.bigint() < deadline) {
    checksum = (checksum + (iterations * 31) % 9973) % 1000003
    iterations++
  }
  res.json({ seconds, iterations, checksum })
})

app.get('/allocate', (req, res) => {
  const megabytes = Math.min(Math.max(Number(req.query.mb || 8), 1), 64)
  allocationHold.push(Buffer.alloc(megabytes * 1024 * 1024, 1))
  if (allocationHold.length > 4) allocationHold.shift()
  res.json({ allocated_mb: megabytes, retained_blocks: allocationHold.length })
})

app.get('/exceptions', (req, res) => {
  const count = Math.min(Math.max(Number(req.query.count || 100), 1), 5000)
  let caught = 0
  for (let index = 0; index < count; index++) {
    try {
      throw new Error(`intentional-profile-exception-${index % 5}`)
    } catch {
      caught++
    }
  }
  res.json({ caught })
})

app.get('/db/query', async (_req, res, next) => {
  try {
    const result = await pool.query(
      'SELECT status, count(*) FROM orders WHERE customer_id = $1 GROUP BY status',
      [42]
    )
    res.json({ rows: result.rows })
  } catch (error) {
    next(error)
  }
})

app.get('/db/slow', async (_req, res, next) => {
  try {
    const result = await pool.query('SELECT pg_sleep(1.25), count(*) FROM orders')
    res.json({ count: result.rows[0].count, slept_seconds: 1.25 })
  } catch (error) {
    next(error)
  }
})

app.get('/db/error', async (_req, res) => {
  try {
    await pool.query('SELECT * FROM validation_table_that_does_not_exist')
    res.status(500).json({ error: 'not raised' })
  } catch (error) {
    console.warn('intentional database validation error', error.code)
    res.status(500).json({ error: error.code })
  }
})

app.use((error, _req, res, _next) => {
  console.error(error)
  res.status(500).json({ error: error.message })
})

const server = app.listen(5000, () => console.log('listening on 5000'))

async function shutdown () {
  server.close()
  await pool.end()
}

process.on('SIGTERM', shutdown)
process.on('SIGINT', shutdown)
