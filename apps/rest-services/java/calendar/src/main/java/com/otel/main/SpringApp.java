/*
Unless explicitly stated otherwise all files in this repository are licensed
under the Apache 2.0 License.

This product includes software developed at Datadog (https://www.datadoghq.com/)
Copyright 2023 Datadog, Inc.
 */
package com.otel.main;

import io.opentelemetry.api.GlobalOpenTelemetry;
import io.opentelemetry.api.OpenTelemetry;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.annotation.Bean;

@SpringBootApplication(scanBasePackages = "com.otel")
public class SpringApp {
  public static void main(String[] args) {
    SpringApplication.run(SpringApp.class, args);
  }
  @Bean
  public OpenTelemetry openTelemetry() {
    return GlobalOpenTelemetry.get();
  }

  @Bean
  public String serviceName() {
    String serviceName = System.getProperty("otel.serviceName");
    return   serviceName != null ? serviceName : "calendar";
  }
}
