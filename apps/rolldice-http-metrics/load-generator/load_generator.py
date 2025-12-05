#!/usr/bin/env python3

import requests
import time
import random
import json
import logging
import os
from datetime import datetime

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# Configuration
GAME_CONTROLLER_URL = os.getenv("GAME_CONTROLLER_URL", "http://game-controller:5002/play_game")
AVERAGE_INTERVAL = float(os.getenv("AVERAGE_INTERVAL", "10.0"))  # Average 10 seconds between requests

# List of player names for variety
PLAYER_NAMES = [
    "Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace", "Henry",
    "Iris", "Jack", "Kate", "Liam", "Mia", "Noah", "Olivia", "Paul",
    "Quinn", "Ruby", "Sam", "Tina", "Uma", "Victor", "Wendy", "Xander",
    "Yara", "Zoe"
]

def poisson_interval(rate):
    """
    Generate intervals following a Poisson process.
    Rate is the average number of events per unit time.
    Returns the time until the next event.
    """
    return random.expovariate(rate)

def make_game_request():
    """Make a single game request to the game controller."""
    player = random.choice(PLAYER_NAMES)
    
    # 20% chance to make an intentionally failing request
    is_error_request = random.random() < 0.20
    
    if is_error_request:
        # Generate different types of errors
        error_type = random.choice([
            "invalid_endpoint",
            "invalid_payload", 
            "missing_payload",
            "invalid_method"
        ])
        
        if error_type == "invalid_endpoint":
            # Hit a non-existent endpoint
            url = GAME_CONTROLLER_URL.replace("/play_game", "/nonexistent_endpoint")
            payload = {"player": player}
        elif error_type == "invalid_payload":
            # Send invalid data
            url = GAME_CONTROLLER_URL
            payload = {"invalid_field": "bad_data", "player": None}
        elif error_type == "missing_payload":
            # Send empty payload
            url = GAME_CONTROLLER_URL
            payload = {}
        else:  # invalid_method
            # Use wrong HTTP method
            url = GAME_CONTROLLER_URL
            payload = {"player": player}
    else:
        # Normal request
        url = GAME_CONTROLLER_URL
        payload = {"player": player}
    
    try:
        start_time = time.time()
        
        if is_error_request and error_type == "invalid_method":
            # Use GET instead of POST
            response = requests.get(url, params=payload, timeout=30)
        else:
            response = requests.post(
                url,
                json=payload,
                headers={"Content-Type": "application/json"},
                timeout=30
            )
        
        end_time = time.time()
        duration = end_time - start_time
        
        if response.status_code == 200:
            result = response.json()
            logger.info(f"✅ Success: Player {player} -> {result} (Duration: {duration:.2f}s)")
            return True
        else:
            if is_error_request:
                logger.info(f"🔥 Intentional Error: Player {player} -> {response.status_code}: {response.text[:100]} (Duration: {duration:.2f}s)")
            else:
                logger.error(f"❌ Unexpected Error: Player {player} -> {response.status_code}: {response.text[:100]} (Duration: {duration:.2f}s)")
            return False
    
    except requests.exceptions.RequestException as e:
        if is_error_request:
            logger.info(f"🔥 Intentional Error: Player {player} -> Request failed: {str(e)[:100]}")
        else:
            logger.error(f"❌ Request failed for player {player}: {e}")
        return False

def main():
    """Main load generation loop."""
    logger.info(f"🚀 Starting load generator")
    logger.info(f"📊 Target: {GAME_CONTROLLER_URL}")
    logger.info(f"⏰ Average interval: {AVERAGE_INTERVAL} seconds")
    logger.info(f"📈 Average rate: {1/AVERAGE_INTERVAL:.3f} requests/second")
    logger.info(f"🔥 Error rate: 20% (intentional errors)")
    
    request_count = 0
    success_count = 0
    error_count = 0
    rate = 1.0 / AVERAGE_INTERVAL  # Convert average interval to rate
    
    while True:
        try:
            # Generate next interval using Poisson process
            next_interval = poisson_interval(rate)
            
            logger.info(f"⏳ Waiting {next_interval:.2f}s until next request...")
            time.sleep(next_interval)
            
            # Make the request
            request_count += 1
            logger.info(f"🎯 Making request #{request_count}")
            success = make_game_request()
            
            if success:
                success_count += 1
            else:
                error_count += 1
            
            # Log statistics every 10 requests
            if request_count % 10 == 0:
                success_rate = (success_count / request_count) * 100
                error_rate = (error_count / request_count) * 100
                logger.info(f"📊 Stats: {request_count} requests, {success_rate:.1f}% success, {error_rate:.1f}% errors")
            
            if not success:
                # Small backoff on failure
                time.sleep(1)
                
        except KeyboardInterrupt:
            logger.info("🛑 Stopping load generator...")
            break
        except Exception as e:
            logger.error(f"❌ Unexpected error: {e}")
            time.sleep(5)  # Backoff on unexpected errors

if __name__ == "__main__":
    main()