const { Pool } = require('pg');

const pool = new Pool({
    user: 'dinesh.gurumurthy',
    host: 'localhost',
    database: 'postgres',
    port: 5432,
});

module.exports = pool;

