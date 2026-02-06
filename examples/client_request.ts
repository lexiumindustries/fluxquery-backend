import * as crypto from 'crypto';
import axios from 'axios';

// This example shows how a Node.js/TypeScript server would securely call the Export API.

async function callExportApi() {
    const secret = 'devsecret';
    const url = 'http://localhost:8080/export';
    const path = '/export';
    const method = 'POST';

    const body = JSON.stringify({
        query: 'SELECT * FROM users LIMIT 10',
        email: 'admin@example.com',
        format: 'json'
    });

    // 1. Generate Authentication Headers
    const timestamp = Math.floor(Date.now() / 1000).toString();

    // Payload for signature: Method + Path + Body + Timestamp
    const payload = method + path + body + timestamp;

    const signature = crypto
        .createHmac('sha256', secret)
        .update(payload)
        .digest('hex');

    // 2. Send Request
    try {
        const response = await axios.post(url, body, {
            headers: {
                'Content-Type': 'application/json',
                'X-Timestamp': timestamp,
                'X-Signature': signature
            }
        });

        console.log('Status:', response.status);
        console.log('Response:', response.data);
    } catch (error: any) {
        if (error.response) {
            console.error('Error Status:', error.response.status);
            console.error('Error Data:', error.response.data);
        } else {
            console.error('Error:', error.message);
        }
    }
}

callExportApi();
