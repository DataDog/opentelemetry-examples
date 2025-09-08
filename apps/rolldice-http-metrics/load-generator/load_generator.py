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
    """Make a single game request to the game controller with 80/20 success/error ratio."""
    # 80% valid requests, 20% error requests (empty body)
    is_error_request = random.random() < 0.2
    
    if is_error_request:
        # Send empty request body to generate error
        payload = {}
        player = "ERROR_REQUEST"
    else:
        # Send valid request
        player = random.choice(PLAYER_NAMES)
        payload = {"player": player}
    
    try:
        start_time = time.time()
        response = requests.post(
            GAME_CONTROLLER_URL,
            json=payload,
            headers={"Content-Type": "application/json"},
            timeout=30
        )
        end_time = time.time()
        duration = end_time - start_time
        
        if is_error_request:
            # Expected error response
            if response.status_code >= 400:
                logger.info(f"🔥 Expected Error: Empty body -> {response.status_code} (Duration: {duration:.2f}s)")
                return True
            else:
                logger.warning(f"⚠️ Unexpected Success: Empty body -> {response.status_code}")
                return True
        else:
            # Expected success response
            if response.status_code == 200:
                result = response.json()
                logger.info(f"✅ Success: Player {player} -> {result} (Duration: {duration:.2f}s)")
                return True
            else:
                logger.error(f"❌ Unexpected Error: Player {player} -> {response.status_code}: {response.text}")
                return False
    
    except requests.exceptions.RequestException as e:
        logger.error(f"❌ Request failed for player {player}: {e}")
        return False

def main():
    """Main load generation loop."""
    logger.info(f"🚀 Starting load generator")
    logger.info(f"📊 Target: {GAME_CONTROLLER_URL}")
    logger.info(f"⏰ Average interval: {AVERAGE_INTERVAL} seconds")
    logger.info(f"📈 Average rate: {1/AVERAGE_INTERVAL:.3f} requests/second")
    
    request_count = 0
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