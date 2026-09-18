'use strict'

function dynamicProbeTarget (value) {
  const normalized = Math.max(0, value)
  const result = normalized * 2 + 1
  return result
}

module.exports = { dynamicProbeTarget }
