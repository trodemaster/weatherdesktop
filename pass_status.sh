#!/bin/bash

# source url
SOURCE_URL="https://wsdot.com/travel/real-time/mountainpasses/stevens"

# selector path
SELECTOR_PATH='#index > div:nth-child(7) > div.full-width.column-container.mountain-pass > div.column-1'

HTML_TEMP_FILE="/Users/blake/scratch/wsdot_stevens_pass.html"

## save selected html to file
shot-scraper html $SOURCE_URL -o "$HTML_TEMP_FILE" --selector "$SELECTOR_PATH"

# HTML_TEMP_FILE=~/scratch/closed_wsdot_stevens_pass.html
# HTML_TEMP_FILE=~/scratch/avalanche_wsdot_stevens_pass.html

# use pup to extract the conditions
PASS_STATUS_EAST=$(cat $HTML_TEMP_FILE | pup 'body > div > div:nth-child(4) > div.conditionValue' text{})
PASS_STATUS_WEST=$(cat $HTML_TEMP_FILE | pup 'body > div > div:nth-child(5) > div.conditionValue' text{})
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

# if the pass is close render the conditions
if [[ $PASS_STATUS == *"Closed"* ]]; then
  PASS_CONDITIONS=$(cat $HTML_TEMP_FILE | pup 'body > div > div:nth-child(6) > div.conditionValue' text{} | sed 's/ \{2,\}/ /g' | tr -d '\n')
  convert -size 250x200 -background white -fill black -pointsize 14 -gravity center -border 5 -bordercolor white caption:"$PASS_CONDITIONS" assets/pass_conditions.png
else
  convert -size 250x200 xc:none assets/pass_conditions.png
fi

