const express = require('express')
const app = express()

app.get("/", (req, res) => {
	res.send("This is the / endpoint")
})

app.get("/error", (req, res) => {
	throw new Error('The server ran into an unhandled error!')
})

app.listen(3000, () => {
	console.log("Application server has started.")
})