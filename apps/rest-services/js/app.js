const express = require('express');
const bodyParser = require('body-parser');
const db = require('./database');

const app = express();
const PORT = 3000;

app.use(bodyParser.json());

// CREATE: Add a new user
app.post('/users', async (req, res) => {
    const { name, email } = req.body;
    try {
        const { rows } = await db.query("INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id", [name, email]);
        res.json({ id: rows[0].id, name, email });
    } catch (err) {
        res.status(500).json(err);
    }
});

// READ: Get all users
app.get('/users', async (req, res) => {
    try {
        const { rows } = await db.query("SELECT * FROM users");
        res.json(rows);
    } catch (err) {
        res.status(500).json(err);
    }
});

// READ: Get user by name
app.get('/users/:name', async (req, res) => {
    try {
        const { rows } = await db.query("SELECT * FROM users WHERE name = $1", [req.params.name]);
        if (!rows.length) {
            return res.status(404).json({ message: "User not found" });
        }
        res.json(rows[0]);
    } catch (err) {
        res.status(500).json(err);
    }
});

// UPDATE: Update user by ID (idempotent)
app.put('/users/:id', async (req, res) => {
    const { name, email } = req.body;
    try {
        const { rowCount } = await db.query("UPDATE users SET name = $1, email = $2 WHERE id = $3", [name, email, req.params.id]);
        if (!rowCount) {
            return res.status(404).json({ message: "User not found" });
        }
        res.json({ id: parseInt(req.params.id, 10), name, email });
    } catch (err) {
        res.status(500).json(err);
    }
});

// DELETE: Delete user by ID
app.delete('/users/:id', async (req, res) => {
    try {
        const { rowCount } = await db.query("DELETE FROM users WHERE id = $1", [req.params.id]);
        if (!rowCount) {
            return res.status(404).json({ message: "User not found" });
        }
        res.status(204).send();
    } catch (err) {
        res.status(500).json(err);
    }
});
// READ: Get user by name
app.get('/select', async (req, res) => {
    try {
        sql = "select * from users where name = 'John'"
        const { rows } = await db.query(sql)
        if (!rows.length) {
            return res.status(404).json({ message: "User not found" });
        }
        res.json(rows[0]);
    } catch (err) {
        res.status(500).json(err);
    }
});
app.listen(PORT, () => {
    console.log(`Server started on http://localhost:${PORT}`);
});

