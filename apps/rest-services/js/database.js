const { Pool } = require('pg');

const pool = new Pool({
    user: 'otel.sample',
    host: 'localhost',
    database: 'postgres',
    port: 5432,
});

module.exports = pool;

