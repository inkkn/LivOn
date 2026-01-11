Execute bellow commands to test the app
```
docker exec -it livon-postgres-1 psql -d livon -U postgres
```
---
```
nkkn@fedora:~/Documents/github.com/inkkn/LivOn$ 
curl -X POST http://localhost:8080/auth/register -H "Content-Type: application/json" -d '{"phone": "+91XXXXXXXXXX"}'
```
```
{"message":"OTP sent successfully"}
```
---
```
nkkn@fedora:~/Documents/github.com/inkkn/LivOn$ 
curl -X POST http://localhost:8080/auth/verify -H "Content-Type: application/json" -d '{"phone": "+91XXXXXXXXXX", "code": "725878"}'
```
```
{
    "created_at":"2026-01-11T05:11:54.912006Z",
    "token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NjgxOTQ3MTQsImlhdCI6MTc2ODEwODMxNCwiaXNzIjoibGl2b35tYmFja2VuZCIsInN1YiI6Iis5MTk2OTA2NTg4NjgifQ.1B_5zo03WSrB6CDYTER2jQiSQqtav9t2LDFyD_3t_I4",
    "user_id":"+91XXXXXXXXXX"
}
```
---
```
nkkn@fedora:~/Documents/github.com/inkkn/LivOn$ 
wscat -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NjgxOTQ3MTQsImlhdCI6MTc2ODEwODMxNCwiaXNzIjoibGl2b24tYmFja2VuZCIsInN1YiI6Iis5MTk2OTA2NTg4NjgifQ.1B_5zo03WSrB6CDYTER2jQiSQqtav9t2LDFyD_3t_I4" -c "ws://localhost:
8080/ws?conv_id=a1b2c3d4-e5f6-47a8-b9c0-112233445566"
```
---
```
{"type":"message.send","client_msg_id":"1f6d2a42-4f8b-4b9a-bb9a-9e3c1c2b8f01","payload":"Hello world"}
{"type":"message.send","client_msg_id":"1f6d2a42-4f8b-4b9a-bb9a-9e3c1c2b8f01","payload":"Hello world"}
{"type":"message.send","client_msg_id":"a7c91a4e-7e5d-41f8-bd92-3b4b91e7f2a4","payload":"How are you?"}
{"type":"message.send","client_msg_id":"e3d84c6a-5f9a-4f63-9e51-29c1c6b7a011","payload":"Check this out 🔥"}
{"type":"message.send","client_msg_id":"c2e9c1b9-33d6-4c55-9f22-5d7b4d3a91a2","payload":"This is a longer message to verify retries and idempotency"}
```
---