package com.otel.main;

import io.opentelemetry.api.GlobalOpenTelemetry;
import io.opentelemetry.api.OpenTelemetry;
import java.lang.management.ManagementFactory;
import javax.management.MBeanServer;
import javax.management.ObjectName;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.annotation.Bean;
import org.springframework.context.event.ContextRefreshedEvent;
import org.springframework.context.event.EventListener;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

@SpringBootApplication(scanBasePackages = "com.otel")
public class SpringApp {

    private static final Logger log = LoggerFactory.getLogger(SpringApp.class);

    public static void main(String[] args) {
        SpringApplication.run(SpringApp.class, args);
    }

    @EventListener(ContextRefreshedEvent.class)
    public void registerMBean() {
        try {
            MBeanServer mBeanServer = ManagementFactory.getPlatformMBeanServer();
            ObjectName objectName = new ObjectName("com.otel.main:type=Calendar");
            Calendar calendarMBean = new Calendar();
            mBeanServer.registerMBean(calendarMBean, objectName);
            log.info("MBean registered: {}", objectName);
        } catch (Exception e) {
            log.error("Error registering MBean", e);
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
    private long totalRequestLatency = 0;

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
        totalRequestLatency += (long) latency;
    }
}

}  
