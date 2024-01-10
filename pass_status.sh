#!/bin/bash

# source url 
SOURCE_URL="https://wsdot.com/travel/real-time/mountainpasses/stevens"

# selector path
SELECTOR_PATH='#index > div:nth-child(7) > div.full-width.column-container.mountain-pass > div.column-1'

# save selected html to file
shot-scraper html $SOURCE_URL -o ~/scratch/wsdot_stevens_pass.html --selector "$SELECTOR_PATH"

# use pup to extract the conditions
PASS_STATUS_EAST=$(cat ~/scratch/wsdot_stevens_pass.html | pup 'body > div > div:nth-child(4) > div.conditionValue' text{})
PASS_STATUS_WEST=$(cat ~/scratch/wsdot_stevens_pass.html | pup 'body > div > div:nth-child(5) > div.conditionValue' text{})

PASS_STATUS="Open"

# if path status contains "Closed" then echo "Closed"
if [[ $PASS_STATUS_EAST == *"Closed"* ]]; then
  PASS_STATUS="Closed"
  echo "East Closed"
else
  echo "East Open"
fi

# if path status contains "Closed" then echo "Closed"
if [[ $PASS_STATUS_WEST == *"Closed"* ]]; then
  PASS_STATUS="Closed"
  echo "West Closed"
else
  echo "West Open"
fi

# if path status contains "Closed" then echo "Closed"
if [[ $PASS_STATUS == *"Closed"* ]]; then
  
else
  echo "Open"
fi