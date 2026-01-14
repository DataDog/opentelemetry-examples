package com.example.gamecontroller.controller;

import java.util.HashMap;
import java.util.Map;
import java.util.Random;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpMethod;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.client.RestTemplate;

@RestController
public class GameController {
    
    private final RestTemplate restTemplate;
    private final Random random = new Random();
    
    @Autowired
    public GameController(RestTemplate restTemplate) {
        this.restTemplate = restTemplate;
    }
    
    @PostMapping("/play_game")
    public ResponseEntity<?> playGame(@RequestBody GameRequest request) {
        // Sleep for a random Gaussian amount (mean=3750ms, stddev=500ms)
        try {
            double sleepTime = random.nextGaussian() * 1000 + 1750;
            // Ensure sleep time is positive
            if (sleepTime > 0) {
                Thread.sleep((long) sleepTime);
            }
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }
        String player = request.getPlayer();
        
        try {
            // Create headers
            HttpHeaders headers = new HttpHeaders();
            headers.set("Content-Type", "application/json");
            
            // Make dice roll request
            String rollingUrl = String.format("http://rolling:5004/rolldice?player=%s", player);
            HttpEntity<String> rollingEntity = new HttpEntity<>(headers);
            ResponseEntity<String> diceRollResult = restTemplate.exchange(
                rollingUrl, HttpMethod.GET, rollingEntity, String.class);
            
            // Make score update request
            Map<String, Object> scoreRequest = new HashMap<>();
            scoreRequest.put("player", player);
            scoreRequest.put("result", diceRollResult.getBody());
            
            HttpEntity<Map<String, Object>> scoringEntity = new HttpEntity<>(scoreRequest, headers);
            ResponseEntity<String> updateScoreResult = restTemplate.exchange(
                "http://scoring:5001/update_score", HttpMethod.POST, scoringEntity, String.class);

            return ResponseEntity.ok(updateScoreResult.getBody());
            
        } catch (Exception error) {
            return ResponseEntity.status(500).body("An error occurred: " + error.getMessage());
        }
    }
    
    public static class GameRequest {
        private String player;
        
        public String getPlayer() {
            return player;
        }
        
        public void setPlayer(String player) {
            this.player = player;
        }
    }
}