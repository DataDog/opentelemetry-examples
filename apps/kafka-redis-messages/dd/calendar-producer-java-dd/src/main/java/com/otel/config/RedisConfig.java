package com.otel.config;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import redis.clients.jedis.Jedis;

@Configuration
public class RedisConfig {

  @Value("${redis.host:localhost}")
  private String redisHost;

  @Value("${redis.port:6379}")
  private int redisPort;

  @Bean
  public Jedis jedis() {
    return new Jedis(redisHost, redisPort);
  }
}
