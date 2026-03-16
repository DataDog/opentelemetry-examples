package com.otel.config;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import redis.clients.jedis.JedisPool;
import redis.clients.jedis.JedisPoolConfig;

/**
 * Redis configuration with connection pooling.
 *
 * DD-specific: The dd-java-agent auto-instruments Jedis, creating spans for
 * every Redis command even when using the OTel API variant.
 *
 * Uses JedisPool for production-ready connection pooling instead of a single
 * Jedis instance.
 */
@Configuration
public class RedisConfig {

  @Value("${redis.host:localhost}")
  private String redisHost;

  @Value("${redis.port:6379}")
  private int redisPort;

  @Bean
  public JedisPool jedisPool() {
    JedisPoolConfig poolConfig = new JedisPoolConfig();
    poolConfig.setMaxTotal(10);
    poolConfig.setMaxIdle(5);
    poolConfig.setMinIdle(2);
    poolConfig.setTestOnBorrow(true);
    return new JedisPool(poolConfig, redisHost, redisPort);
  }
}
