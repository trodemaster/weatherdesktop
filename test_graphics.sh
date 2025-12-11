#!/bin/bash

# Test script to verify pass status graphic selection

echo "Testing Stevens Pass graphic selection..."
echo ""

# Test 1: Both closed
echo "Test 1: Both directions closed"
cp testfiles/closed_wsdot_stevens_pass_2025_12_10_rain.html assets/wsdot_stevens_pass.html
./wd -r --debug 2>&1 | grep -E "(Pass Status|Pass status graphic)"
md5sum assets/pass_conditions.png graphics/hw2_closed.png | awk '{print $2 ": " $1}'
echo ""

# Test 2: East closed only
echo "Test 2: East closed only"
cp testfiles/closed_east_wsdot_stevens_pass.html assets/wsdot_stevens_pass.html
./wd -r --debug 2>&1 | grep -E "(Pass Status|Pass status graphic)"
md5sum assets/pass_conditions.png graphics/hw2_closed_e.png | awk '{print $2 ": " $1}'
echo ""

# Test 3: West closed only
echo "Test 3: West closed only"
cp testfiles/closed_west_wsdot_stevens_pass.html assets/wsdot_stevens_pass.html
./wd -r --debug 2>&1 | grep -E "(Pass Status|Pass status graphic)"
md5sum assets/pass_conditions.png graphics/hw2_closed_w.png | awk '{print $2 ": " $1}'
echo ""

# Test 4: Open
echo "Test 4: Pass open"
cp testfiles/open_wsdot_stevens_pass_2024_01_10.html assets/wsdot_stevens_pass.html
./wd -r --debug 2>&1 | grep -E "(Pass Status|Pass status graphic)"
md5sum assets/pass_conditions.png graphics/hw2_open.png | awk '{print $2 ": " $1}'
echo ""

echo "Test complete!"
