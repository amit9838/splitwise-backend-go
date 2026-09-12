# Expense Manager — API Documentation

Complete reference for the frontend team to integrate with the Expense Manager backend.

---

## Table of Contents

1. [Base Information](#base-information)
2. [Authentication](#authentication)
3. [Auth Endpoints](#auth-endpoints)
4. [Users](#users)
5. [Categories](#categories)
6. [Groups](#groups)
7. [Expenses](#expenses)
8. [Balances](#balances)
9. [Settlements](#settlements)
10. [Data Types & Conventions](#data-types--conventions)
11. [Error Responses](#error-responses)

---

## Base Information

| Item | Value |
|------|-------|
| Base URL | `http://<host>:<port>` (e.g. `http://localhost:8000`) |
| Content Type | `application/json` |
| Authentication | Bearer JWT (see [Authentication](#authentication)) |
| CORS | All origins allowed (`*`) |
| Interactive Docs | `/docs` (Swagger UI), `/redoc` (ReDoc) |

All request/response bodies are JSON. IDs are UUIDs serialized as strings. Decimal amounts are serialized as strings (e.g. `"100.50"`).

---

## Authentication

Most endpoints require authentication. Send the access token in the `Authorization` header:

```
Authorization: Bearer <access_token>
```

### Token Types

| Token | Purpose | Expiry |
|-------|---------|--------|
| `access_token` | Authenticate API requests | 30 minutes |
| `refresh_token` | Obtain a new access token | 7 days |

### Auth Flow

1. Register or login to get `access_token` + `refresh_token`.
2. Send `access_token` as Bearer token on protected requests.
3. When the access token expires, call `POST /api/auth/refresh` with the `refresh_token` to get a new pair.

**Public endpoints (no auth required):**
- `POST /api/auth/register`
- `POST /api/auth/login`
- `POST /api/auth/refresh`

All other endpoints require a valid access token.

---

## Auth Endpoints

### `POST /api/auth/register`

Register a new user. **No auth required.**

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "secret123",
  "full_name": "John Doe"
}
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `email` | string | ✅ | Valid email format |
| `password` | string | ✅ | Plain text (hashed server-side) |
| `full_name` | string | ❌ | Optional |

**Response `201 Created`:**

```json
{
  "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "email": "user@example.com",
  "full_name": "John Doe",
  "is_active": true,
  "created_at": "2026-08-16T00:00:00Z",
  "updated_at": null
}
```

**Errors:**
- `400` — `Email already registered`

---

### `POST /api/auth/login`

Authenticate and receive tokens. **No auth required.**

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "secret123"
}
```

| Field | Type | Required |
|-------|------|----------|
| `email` | string | ✅ |
| `password` | string | ✅ |

**Response `200 OK`:**

```json
{
  "access_token": "eyJhbGciOi...",
  "refresh_token": "eyJhbGciOi...",
  "token_type": "bearer"
}
```

**Errors:**
- `401` — `Invalid email or password`
- `403` — `Account is deactivated`

---

### `POST /api/auth/refresh`

Exchange a refresh token for a new token pair. **No auth required.**

**Request Body:**

```json
{
  "refresh_token": "eyJhbGciOi..."
}
```

| Field | Type | Required |
|-------|------|----------|
| `refresh_token` | string | ✅ |

**Response `200 OK`:**

```json
{
  "access_token": "eyJhbGciOi...",
  "refresh_token": "eyJhbGciOi...",
  "token_type": "bearer"
}
```

**Errors:**
- `401` — `Invalid token type` / `Invalid token` / `Invalid or expired refresh token` / `User not found or inactive`

---

### `GET /api/auth/me`

Get the currently authenticated user. **Auth required.**

**Response `200 OK`:**

```json
{
  "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "email": "user@example.com",
  "full_name": "John Doe",
  "is_active": true,
  "created_at": "2026-08-16T00:00:00Z",
  "updated_at": null
}
```

---

## Users

### `GET /api/users/`

List all users. **Auth required.**

**Response `200 OK`:**

```json
[
  {
    "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "email": "user@example.com",
    "full_name": "John Doe",
    "is_active": true,
    "created_at": "2026-08-16T00:00:00Z",
    "updated_at": null
  }
]
```

---

### `GET /api/users/{user_id}`

Get a single user by ID. **Auth required.**

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `user_id` | UUID | User ID |

**Response `200 OK`:** Same shape as a single user object (see above).

**Errors:**
- `404` — `User not found`

---

### `POST /api/users/`

Create a user. **No auth required.**

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "secret123",
  "full_name": "John Doe"
}
```

| Field | Type | Required |
|-------|------|----------|
| `email` | string | ✅ |
| `password` | string | ✅ |
| `full_name` | string | ❌ |

**Response `201 Created`:** Single user object.

**Errors:**
- `400` — `Email already registered`

---

### `PUT /api/users/{user_id}`

Update a user. **Auth required.** All fields optional; only provided fields are updated.

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `user_id` | UUID | User ID |

**Request Body:**

```json
{
  "email": "new@example.com",
  "full_name": "Jane Doe",
  "password": "newpass123",
  "is_active": true
}
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `email` | string | ❌ | Must be unique |
| `full_name` | string | ❌ | |
| `password` | string | ❌ | Will be re-hashed |
| `is_active` | boolean | ❌ | |

**Response `200 OK`:** Updated user object.

**Errors:**
- `404` — `User not found`
- `400` — `Email already registered`

---

### `DELETE /api/users/{user_id}`

Delete a user. **Auth required.**

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `user_id` | UUID | User ID |

**Response `200 OK`:**

```json
"User deleted successfully!"
```

**Errors:**
- `404` — `User not found`

---

## Categories

### `GET /api/categories/`

List all categories. **Auth required.**

**Response `200 OK`:**

```json
[
  {
    "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "name": "Food",
    "is_active": true,
    "created_at": "2026-08-16T00:00:00Z"
  }
]
```

---

### `GET /api/categories/{category_id}`

Get a single category. **Auth required.**

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `category_id` | UUID | Category ID |

**Response `200 OK`:** Single category object.

**Errors:**
- `404` — `Category not found`

---

### `POST /api/categories/`

Create a category. **Auth required.**

**Request Body:**

```json
{
  "name": "Food"
}
```

| Field | Type | Required |
|-------|------|----------|
| `name` | string | ✅ |

**Response `201 Created`:** Single category object.

**Errors:**
- `400` — `Category already exists!`

---

### `PUT /api/categories/{category_id}`

Update a category. **Auth required.** All fields optional.

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `category_id` | UUID | Category ID |

**Request Body:**

```json
{
  "name": "Groceries",
  "is_active": true
}
```

| Field | Type | Required |
|-------|------|----------|
| `name` | string | ❌ |
| `is_active` | boolean | ❌ |

**Response `200 OK`:** Updated category object.

**Errors:**
- `404` — `Category not found`

---

### `DELETE /api/categories/{category_id}`

Delete a category. **Auth required.**

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `category_id` | UUID | Category ID |

**Response `200 OK`:**

```json
"Category Food deleted successfully!"
```

**Errors:**
- `404` — `Category not found`

---

## Groups

### `GET /api/groups/`

List groups the current user is an active member of. **Auth required.**

**Response `200 OK`:**

```json
[
  {
    "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "name": "Trip to Goa",
    "created_by": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "simplify_debts": true,
    "is_active": true,
    "created_at": "2026-08-16T00:00:00Z",
    "updated_at": null,
    "members": [
      {
        "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
        "user_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
        "user": {
          "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
          "email": "user@example.com",
          "full_name": "John Doe"
        },
        "joined_at": "2026-08-16T00:00:00Z",
        "is_active": true
      }
    ]
  }
]
```

---

### `GET /api/groups/{group_id}`

Get a single group with members. **Auth required.**

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `group_id` | UUID | Group ID |

**Response `200 OK`:** Single group object (see above).

**Errors:**
- `404` — `Group not found`

---

### `POST /api/groups/`

Create a group. **Auth required.** The creator is automatically added as a member.

**Request Body:**

```json
{
  "name": "Trip to Goa",
  "member_ids": [
    "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "4fa85f64-5717-4562-b3fc-2c963f66afa6"
  ],
  "simplify_debts": true
}
```

| Field | Type | Required | Default | Notes |
|-------|------|----------|---------|-------|
| `name` | string | ✅ | — | |
| `member_ids` | UUID[] | ❌ | `[]` | Other members to add. Creator must NOT be included. |
| `simplify_debts` | boolean | ❌ | `true` | Enable debt simplification |

**Response `201 Created`:** Single group object with members.

**Errors:**
- `400` — `Duplicate member ids provided`
- `400` — `Creator is added automatically and must not be in member_ids`
- `404` — `One or more members not found`

---

### `PUT /api/groups/{group_id}`

Update a group. **Auth required.** All fields optional.

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `group_id` | UUID | Group ID |

**Request Body:**

```json
{
  "name": "Goa Trip 2026",
  "simplify_debts": false,
  "is_active": true
}
```

| Field | Type | Required |
|-------|------|----------|
| `name` | string | ❌ |
| `simplify_debts` | boolean | ❌ |
| `is_active` | boolean | ❌ |

**Response `200 OK`:** Updated group object.

**Errors:**
- `404` — `Group not found`

---

### `DELETE /api/groups/{group_id}`

Delete a group. **Auth required.** Only the group creator can delete.

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `group_id` | UUID | Group ID |

**Response `200 OK`:**

```json
"Group 'Trip to Goa' deleted successfully!"
```

**Errors:**
- `404` — `Group not found`
- `403` — `Only group creator can delete the group`

---

### `POST /api/groups/{group_id}/members`

Add a member to a group. **Auth required.**

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `group_id` | UUID | Group ID |

**Request Body:**

```json
{
  "user_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6"
}
```

| Field | Type | Required |
|-------|------|----------|
| `user_id` | UUID | ✅ |

**Response `200 OK`:** Updated group object with members.

**Errors:**
- `404` — `Group not found`
- `404` — `User not found`
- `400` — `User is already a member of this group`

---

### `DELETE /api/groups/{group_id}/members/{user_id}`

Remove a member from a group. **Auth required.** Only the group creator can remove members.

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `group_id` | UUID | Group ID |
| `user_id` | UUID | Member user ID to remove |

**Response `200 OK`:**

```json
"Member removed successfully!"
```

**Errors:**
- `404` — `Group not found`
- `403` — `Only group creator can remove members`
- `400` — `Cannot remove the group creator`
- `404` — `Member not found in group`

---

## Expenses

### Split Types

Expenses support four split strategies, specified via the `split_type` field:

| `split_type` | Description | `splits` array required? |
|--------------|-------------|--------------------------|
| `EQUAL` | Split evenly among all active group members | ❌ (ignored) |
| `EXACT` | Each split specifies an exact `amount` | ✅ |
| `PERCENTAGE` | Each split specifies a `percentage` (must sum to 100) | ✅ |
| `SHARES` | Each split specifies a `shares` count (proportional) | ✅ |

The `splits` array entries have this shape:

```json
{
  "user_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "amount": "50.00",
  "percentage": 25.0,
  "shares": 1
}
```

| Field | Type | Used by |
|-------|------|---------|
| `user_id` | UUID | All non-EQUAL types |
| `amount` | decimal | `EXACT` |
| `percentage` | float | `PERCENTAGE` |
| `shares` | integer | `SHARES` |

---

### `GET /api/expenses/group/{group_id}`

List active expenses for a group. **Auth required.**

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `group_id` | UUID | Group ID |

**Response `200 OK`:**

```json
[
  {
    "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "group_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "paid_by": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "category_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "amount": "150.00",
    "description": "Dinner",
    "currency": "INR",
    "split_type": "EQUAL",
    "expense_date": "2026-08-16T00:00:00Z",
    "is_active": true,
    "created_at": "2026-08-16T00:00:00Z",
    "updated_at": null,
    "splits": [
      {
        "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
        "user_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
        "amount": "50.00",
        "percentage": null,
        "shares": null
      }
    ]
  }
]
```

---

### `GET /api/expenses/{expense_id}`

Get a single expense. **Auth required.**

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `expense_id` | UUID | Expense ID |

**Response `200 OK`:** Single expense object (see above).

**Errors:**
- `404` — `Expense not found`

---

### `POST /api/expenses/`

Create an expense. **Auth required.** The `paid_by` field is always set to the authenticated user. Splits are computed server-side based on `split_type`.

**Request Body (EQUAL split):**

```json
{
  "group_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "category_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "amount": "150.00",
  "description": "Dinner",
  "currency": "INR",
  "split_type": "EQUAL",
  "expense_date": "2026-08-16T00:00:00Z",
  "splits": []
}
```

**Request Body (EXACT split):**

```json
{
  "group_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "category_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "amount": "150.00",
  "description": "Dinner",
  "currency": "INR",
  "split_type": "EXACT",
  "expense_date": "2026-08-16T00:00:00Z",
  "splits": [
    { "user_id": "aaa...", "amount": "100.00" },
    { "user_id": "bbb...", "amount": "50.00" }
  ]
}
```

**Request Body (PERCENTAGE split):**

```json
{
  "group_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "category_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "amount": "150.00",
  "description": "Dinner",
  "currency": "INR",
  "split_type": "PERCENTAGE",
  "expense_date": "2026-08-16T00:00:00Z",
  "splits": [
    { "user_id": "aaa...", "percentage": 60.0 },
    { "user_id": "bbb...", "percentage": 40.0 }
  ]
}
```

**Request Body (SHARES split):**

```json
{
  "group_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "category_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "amount": "150.00",
  "description": "Dinner",
  "currency": "INR",
  "split_type": "SHARES",
  "expense_date": "2026-08-16T00:00:00Z",
  "splits": [
    { "user_id": "aaa...", "shares": 2 },
    { "user_id": "bbb...", "shares": 1 }
  ]
}
```

| Field | Type | Required | Default | Notes |
|-------|------|----------|---------|-------|
| `group_id` | UUID | ✅ | — | |
| `category_id` | UUID | ✅ | — | Must exist |
| `amount` | decimal | ✅ | — | Must be > 0 |
| `description` | string | ✅ | — | |
| `currency` | string | ❌ | `"INR"` | |
| `split_type` | string | ❌ | `"EQUAL"` | `EQUAL`, `EXACT`, `PERCENTAGE`, `SHARES` |
| `expense_date` | datetime | ✅ | — | |
| `splits` | array | ❌ | `[]` | See split types |

**Response `201 Created`:** Single expense object with computed splits.

**Errors:**
- `404` — `Group not found`
- `400` — `Category not found`
- `400` — `Group has no members`
- `400` — Various split validation errors (e.g. `Sum of exact splits (...) must equal total (...)`, `Percentages must sum to 100`, `Total shares must be greater than 0`, `Unknown split type`)

---

### `PUT /api/expenses/{expense_id}`

Update an expense. **Auth required.** All fields optional. Note: updating `amount` or `split_type` does NOT recompute splits.

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `expense_id` | UUID | Expense ID |

**Request Body:**

```json
{
  "category_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "amount": "200.00",
  "description": "Updated dinner",
  "currency": "INR",
  "split_type": "EQUAL",
  "expense_date": "2026-08-16T00:00:00Z",
  "is_active": true
}
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `category_id` | UUID | ❌ | Must exist if provided |
| `amount` | decimal | ❌ | Must be > 0 |
| `description` | string | ❌ | |
| `currency` | string | ❌ | |
| `split_type` | string | ❌ | |
| `expense_date` | datetime | ❌ | |
| `is_active` | boolean | ❌ | |

**Response `200 OK`:** Updated expense object.

**Errors:**
- `404` — `Expense not found`
- `400` — `Category not found`

---

### `DELETE /api/expenses/{expense_id}`

Delete an expense. **Auth required.**

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `expense_id` | UUID | Expense ID |

**Response `200 OK`:**

```json
"Expense deleted successfully!"
```

**Errors:**
- `404` — `Expense not found`

---

## Balances

### `GET /api/balances/group/{group_id}`

Get raw balances and simplified settlements for a group. **Auth required.**

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `group_id` | UUID | Group ID |

**Response `200 OK`:**

```json
{
  "group_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "group_name": "Trip to Goa",
  "balances": [
    {
      "user_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
      "email": "user@example.com",
      "full_name": "John Doe",
      "net_balance": 50.0,
      "status": "owed"
    },
    {
      "user_id": "4fa85f64-5717-4562-b3fc-2c963f66afa6",
      "email": "jane@example.com",
      "full_name": "Jane Doe",
      "net_balance": -50.0,
      "status": "owes"
    }
  ],
  "simplified_settlements": [
    {
      "from_user_id": "4fa85f64-5717-4562-b3fc-2c963f66afa6",
      "to_user_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
      "amount": 50.0
    }
  ],
  "total_settlements": 1
}
```

**Balance `status` values:**
- `owed` — positive balance (others owe this user)
- `owes` — negative balance (this user owes others)
- `settled` — zero balance

**Errors:**
- `404` — `Group not found`

---

### `GET /api/balances/me`

Get balances across all groups for the current user. **Auth required.**

**Response `200 OK`:**

```json
{
  "user_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "total_owed_to_me": 120.0,
  "total_i_owe": 30.0,
  "net_balance": 90.0,
  "groups": [
    {
      "group_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
      "group_name": "Trip to Goa",
      "my_net_balance": 50.0,
      "my_settlements": [
        {
          "from_user_id": "4fa85f64-5717-4562-b3fc-2c963f66afa6",
          "to_user_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
          "amount": 50.0
        }
      ]
    }
  ]
}
```

---

## Settlements

### `GET /api/settlements/group/{group_id}`

List settlements for a group (most recent first). **Auth required.**

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `group_id` | UUID | Group ID |

**Response `200 OK`:**

```json
[
  {
    "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "group_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "paid_by": "4fa85f64-5717-4562-b3fc-2c963f66afa6",
    "paid_to": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "amount": "50.00",
    "payment_method": "cash",
    "note": "Dinner split",
    "settled_at": "2026-08-16T00:00:00Z",
    "created_at": "2026-08-16T00:00:00Z"
  }
]
```

---

### `POST /api/settlements/`

Record a settlement. **Auth required.** The `paid_by` field is always the authenticated user.

**Request Body:**

```json
{
  "group_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "paid_to": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "amount": "50.00",
  "payment_method": "cash",
  "note": "Dinner split"
}
```

| Field | Type | Required | Default | Notes |
|-------|------|----------|---------|-------|
| `group_id` | UUID | ✅ | — | |
| `paid_to` | UUID | ✅ | — | Recipient; must be an active group member |
| `amount` | decimal | ✅ | — | Must be > 0 |
| `payment_method` | string | ❌ | `"cash"` | |
| `note` | string | ❌ | `null` | |

**Response `201 Created`:** Single settlement object.

**Errors:**
- `404` — `Group not found`
- `400` — `Recipient is not a member of this group`
- `400` — `Cannot settle with yourself`
- `400` — `Settlement amount must be positive`

---

### `DELETE /api/settlements/{settlement_id}`

Delete a settlement. **Auth required.** Only the payer can delete.

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `settlement_id` | UUID | Settlement ID |

**Response `200 OK`:**

```json
"Settlement deleted successfully!"
```

**Errors:**
- `404` — `Settlement not found`
- `403` — `Only the payer can delete a settlement`

---

## Data Types & Conventions

| Type | JSON Representation | Notes |
|------|---------------------|-------|
| UUID | string | e.g. `"3fa85f64-5717-4562-b3fc-2c963f66afa6"` |
| Decimal | string | e.g. `"150.00"` — always serialized as string |
| datetime | ISO 8601 string | e.g. `"2026-08-16T00:00:00Z"` |
| boolean | boolean | `true` / `false` |
| float | number | e.g. `50.0` (used in balances) |

---

## Error Responses

Errors are returned as JSON with a `detail` field:

```json
{
  "detail": "Error message here"
}
```

### Common Status Codes

| Status | Meaning |
|--------|---------|
| `200` | Success |
| `201` | Resource created |
| `400` | Bad request / validation error |
| `401` | Unauthorized (missing/invalid/expired token) |
| `403` | Forbidden (insufficient permissions) |
| `404` | Resource not found |
| `422` | Request validation error (FastAPI/Pydantic) |

### Authentication Errors

- `401` — `Invalid token type` (wrong token used)
- `401` — `Invalid token`
- `401` — `Invalid or expired token`
- `401` — `User not found`
- `403` — `Account is deactivated`

---

## Quick Reference — Endpoint Summary

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/auth/register` | ❌ | Register user |
| POST | `/api/auth/login` | ❌ | Login, get tokens |
| POST | `/api/auth/refresh` | ❌ | Refresh tokens |
| GET | `/api/auth/me` | ✅ | Current user |
| GET | `/api/users/` | ✅ | List users |
| GET | `/api/users/{id}` | ✅ | Get user |
| POST | `/api/users/` | ❌ | Create user |
| PUT | `/api/users/{id}` | ✅ | Update user |
| DELETE | `/api/users/{id}` | ✅ | Delete user |
| GET | `/api/categories/` | ✅ | List categories |
| GET | `/api/categories/{id}` | ✅ | Get category |
| POST | `/api/categories/` | ✅ | Create category |
| PUT | `/api/categories/{id}` | ✅ | Update category |
| DELETE | `/api/categories/{id}` | ✅ | Delete category |
| GET | `/api/groups/` | ✅ | List my groups |
| GET | `/api/groups/{id}` | ✅ | Get group |
| POST | `/api/groups/` | ✅ | Create group |
| PUT | `/api/groups/{id}` | ✅ | Update group |
| DELETE | `/api/groups/{id}` | ✅ | Delete group |
| POST | `/api/groups/{id}/members` | ✅ | Add member |
| DELETE | `/api/groups/{id}/members/{user_id}` | ✅ | Remove member |
| GET | `/api/expenses/group/{group_id}` | ✅ | List group expenses |
| GET | `/api/expenses/{id}` | ✅ | Get expense |
| POST | `/api/expenses/` | ✅ | Create expense |
| PUT | `/api/expenses/{id}` | ✅ | Update expense |
| DELETE | `/api/expenses/{id}` | ✅ | Delete expense |
| GET | `/api/balances/group/{group_id}` | ✅ | Group balances |
| GET | `/api/balances/me` | ✅ | My balances |
| GET | `/api/settlements/group/{group_id}` | ✅ | List settlements |
| POST | `/api/settlements/` | ✅ | Record settlement |
| DELETE | `/api/settlements/{id}` | ✅ | Delete settlement |