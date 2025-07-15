## 🐛 Bug Report: REST Backend Server Endpoint

### Issue Title (Name issue with ticket number, priority level and title)

  * ` [DYS-1904][MEDIUM] Bug in  `GET /api/orders`  - Unexpected Sorting Behavior `
  * ` [DYS-2391][HIGH] Bug in  `POST /api/inventorys/allocate`  - Suspected Memory Leaked Endpoint `
  * ` [DYS-0489][CRITICAL] Bug in  `PUT /api/credits`  - Invalid Credit Calculating and Stamping Logic `
  * ` [DYS-1015][HIGH] Bug in  `POST /api/coupons/import`  - API Takes around 20 seconds to response. `
  * ` [DYS-5647][LOW] Bug in  `DELETE /api/items/{id}`  - Minor Logging Inconsistency `

-----

### Issue Details

**Endpoint:** `POST /api/coupons`

**Request Payload (if applicable):**

```json
{
  "coupon_file_key": "coupons/raw_uploads/cc11a7b1-c7b3-499c-a71b-964cc5063579/a311ca29-e95c-47fb-864a-10c93444efb0/coupon_upload_20250714_113510.csv"
}
```

**Steps to Reproduce:**

1.  Send a POST request to `https://your-backend-api.com/api/coupons` with the provided request payload.
2.  Observe the response time.

**Expected Behavior:**
The API should respond within 1-2 seconds, confirming the successful creation of the coupon.

**Current Behavior:**
The API takes approximately 20 seconds to respond to the request. During this time, the client often times out or experiences significant delays. No error message is returned; the coupon is eventually created successfully.

**Logs:**

```
<Please paste relevant server logs here from the time of the slow request. Look for logs related to the `/api/coupons` endpoint, database queries, or any warnings/errors that might indicate a performance bottleneck.>
```

**Additional Context:**
This issue seems to occur consistently when creating new coupons, regardless of the complexity of the coupon data. This significantly impacts the usability of the coupon management system. It has been observed in the Staging environment.

```
```