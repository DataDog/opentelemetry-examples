/*
Unless explicitly stated otherwise all files in this repository are licensed
under the Apache 2.0 License.

This product includes software developed at Datadog (https://www.datadoghq.com/)
Copyright 2023 Datadog, Inc.
 */
package com.otel.main;

import io.opentelemetry.api.GlobalOpenTelemetry;
import io.opentelemetry.api.OpenTelemetry;
import java.lang.management.ManagementFactory;
import javax.management.MBeanServer;
import javax.management.ObjectName;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.annotation.Bean;
import org.springframework.context.event.ContextRefreshedEvent;
import org.springframework.context.event.EventListener;

@SpringBootApplication(scanBasePackages = "com.otel")
public class SpringApp {

    private static final Logger log = LoggerFactory.getLogger(SpringApp.class);

    public static void main(String[] args) {
        SpringApplication.run(SpringApp.class, args);
    }

    @Bean
    public MBeanServer mBeanServer() {
        return ManagementFactory.getPlatformMBeanServer();
    }

    @Bean
    public CalendarMBean calendarMBean() {
        return new Calendar();
    }

    @EventListener(ContextRefreshedEvent.class)
    public void registerAndInitMBean() {
        try {
            MBeanServer mBeanServer = mBeanServer();
            ObjectName objectName = new ObjectName("com.otel.main:type=Calendar");
            Calendar calendarMBean = new Calendar();
            mBeanServer.registerMBean(calendarMBean, objectName);
            log.info("MBean registered: {}", objectName);
        } catch (Exception e) {
            log.error("Error registering and initializing MBean", e);
            throw new IllegalStateException("Failed to register and initialize CalendarMBean", e);
        }
    }

    @Bean
    public OpenTelemetry openTelemetry() {
        return GlobalOpenTelemetry.get();
    }

    @Bean
    public String serviceName() {
        String serviceName = System.getProperty("otel.serviceName");
        return serviceName != null ? serviceName : "calendar";
    }

    public interface CalendarMBean {
        int getHitsCount();
        void incrementHitsCount();
        float getRequestLatency();
        void addRequestLatency(float latency);
    }

    public static class Calendar implements CalendarMBean {
        private int hitsCount = 0;
        private float totalRequestLatency = 0;

        @Override
        public synchronized int getHitsCount() {
            return hitsCount;
        }

        @Override
        public synchronized void incrementHitsCount() {
            hitsCount++;
        }

        @Override
        public synchronized float getRequestLatency() {
            return hitsCount == 0 ? 0 : (float) totalRequestLatency / hitsCount;
        }

        @Override
        public synchronized void addRequestLatency(float latency) {
            totalRequestLatency += latency;
        }
    }
}
