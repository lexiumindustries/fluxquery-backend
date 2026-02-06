<?php

// This example shows how a PHP server would securely call the Export API.

$secret = 'devsecret';
$url = 'http://localhost:8080/export';
$path = '/export';
$method = 'POST';

$data = [
    'query' => 'SELECT * FROM users LIMIT 10',
    'email' => 'admin@example.com',
    'format' => 'json'
];

$body = json_encode($data);

// 1. Generate Authentication Headers
$timestamp = (string)time();

// Payload for signature: Method + Path + Body + Timestamp
$payload = $method . $path . $body . $timestamp;

$signature = hash_hmac('sha256', $payload, $secret);

// 2. Send Request using cURL
$ch = curl_init($url);

curl_setopt($ch, CURLOPT_POST, 1);
curl_setopt($ch, CURLOPT_POSTFIELDS, $body);
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
curl_setopt($ch, CURLOPT_HTTPHEADER, [
    'Content-Type: application/json',
    'X-Timestamp: ' . $timestamp,
    'X-Signature: ' . $signature
]);

$response = curl_exec($ch);
$status = curl_getinfo($ch, CURLINFO_HTTP_CODE);

if (curl_errno($ch)) {
    echo 'Error: ' . curl_error($ch) . "\n";
} else {
    echo "Status: $status\n";
    echo "Response: $response\n";
}

curl_close($ch);
