# INF 395 – Institutional Information System
## Geospatial Institutional Access & Monitoring System (RBAC + H3)

This project was developed for **INF 395 – 1st Assignment (Phase 2)**.

It implements a large-scale institutional backend system that integrates:

- Role-Based Access Control (RBAC)
- H3 geospatial indexing
- Event-driven simulation
- Audit logging
- Institutional constraints handling
- Modular backend architecture

Domain Example Used: **Urban Infrastructure Monitoring System**

---

# 1. How to Run the System

## ✅ Requirements

- Go 1.21+
- Git
- SQLite (or configured SQL database)
- Postman or curl (for testing endpoints)

---

## 🚀 Step 1: Clone Repository

```bash
git clone <your-repository-link>
cd go-rbac-h3-sim-based-on-assignment
```

---

## 🚀 Step 2: Install Dependencies

```bash
go mod tidy
```

---

## 🚀 Step 3: Run Application

```bash
go run main.go
```

OR (if using cmd folder):

```bash
go run ./cmd/server
```

Server runs at:

```
http://localhost:8080
```

---

## 🧪 Example Endpoint Test

```bash
curl -X POST http://localhost:8080/incidents \
-H "Content-Type: application/json" \
-d '{
  "latitude": 43.238949,
  "longitude": 76.889709,
  "type": "infrastructure_failure"
}'
```

---

# 2. Group Member Roles

### 👨‍💻 Member 1 – System Architecture & Backend Lead
- Defined institutional problem
- Designed monolith architecture
- Implemented core REST endpoints
- Designed modular structure
- Implemented RBAC enforcement

### 🌍 Member 2 – H3 & Data Modeling Engineer
- Integrated Uber H3 library
- Designed H3-based region partitioning
- Added H3 index column to database
- Implemented H3-based filtering and aggregation
- Designed ER diagram and logical schema

### 🔐 Member 3 – Security & Event Simulation
- Designed permission matrix
- Implemented ABAC rule
- Implemented audit logging
- Developed event-driven simulation component
- Identified system vulnerability and mitigation

---

# 3. Institutional Problem

Urban institutions must monitor infrastructure failures across city regions.

Challenges:
- Large geographic coverage
- Multiple administrative roles
- Sensitive incident data
- Regional authority boundaries
- Real-time event handling

The system ensures:
- Region-based access control
- Proper logging of actions
- Secure handling of institutional data

---

# 4. Architecture Overview

Architecture Type: **Monolith**

Justification:
- Simpler deployment for institutional environment
- Centralized control
- Easier audit compliance

Structure:

```
Handler Layer (REST API)
        ↓
Service Layer (Business Logic)
        ↓
RBAC + ABAC Validation
        ↓
H3 Spatial Processing
        ↓
Repository Layer (Database)
        ↓
Audit Log
```

Event-Driven Component:
- Incident creation triggers simulated processing event
- Event updates aggregated statistics table

Failure Handling:
- Role validation before execution
- Graceful error responses
- Audit entry for failed attempts

---

# 5. Data Modeling

Includes:

- ER Diagram (conceptual)
- Logical schema
- Audit log table
- Aggregated analytics table
- H3 index column inside `incidents` table

Example Table (simplified):

```
incidents (
    id INTEGER PRIMARY KEY,
    type TEXT,
    latitude REAL,
    longitude REAL,
    h3_index TEXT,
    created_by TEXT,
    created_at TIMESTAMP
)
```

H3 index column enables:
- Region partitioning
- Fast filtering
- Aggregation by hex cell
- Institutional zoning

---

# 6. Security & IAM Design

Roles:

- SuperAdmin
- RegionalManager
- FieldOfficer
- Analyst

Permission Matrix (Simplified):

| Action | SuperAdmin | RegionalManager | FieldOfficer | Analyst |
|--------|------------|----------------|--------------|----------|
| Create Incident | ✔ | ✔ | ✔ | ✖ |
| View All Incidents | ✔ | ✖ | ✖ | ✖ |
| View Regional | ✔ | ✔ | ✖ | ✔ |
| Delete | ✔ | ✖ | ✖ | ✖ |

ABAC Rule Example:

RegionalManager can only view incidents where:
```
user.region_h3 == incident.h3_index (prefix match)
```

Identified Vulnerability:
- Potential IDOR if incident ID is accessed without region check
Mitigation:
- Always validate role AND H3 region ownership before query

---

# 7. How H3 Is Used in the System

H3 is integrated using Uber's H3 Go library.

### 1️⃣ Location to H3 Conversion

When incident is created:

```go
cell := h3.LatLngToCell(lat, lng, resolution)
```

This converts coordinates into a hexagonal cell ID.

---

### 2️⃣ Institutional Zoning

Each region is represented by:
- A list of H3 cells
OR
- A parent H3 index prefix

Access is validated based on H3 comparison.

---

### 3️⃣ Aggregation & Analytics

Incidents can be grouped by:

```
GROUP BY h3_index
```

This enables:
- Regional reporting
- Load balancing
- Heatmap visualization
- Institutional performance metrics

---

### 4️⃣ Why H3 Is Appropriate for This Domain

H3 provides:

- Hexagonal uniform spatial partitioning
- Fast lookup
- Scalable geographic filtering
- Simplified region management
- Efficient aggregation

It is more efficient than raw coordinate comparisons.

---

# 8. Event-Driven Simulation

When a new incident is created:

1. Event is triggered.
2. Aggregated table is updated.
3. Audit log entry is created.

This simulates institutional processing workflow.

---

# 9. Endpoints (Minimum 5 Implemented)

- POST /incidents
- GET /incidents
- GET /incidents/{id}
- DELETE /incidents/{id}
- GET /analytics/region
- GET /audit-logs

RBAC is enforced at handler/service level.

---

# 10. Audit Logging

Audit table logs:

- Incident creation
- Incident deletion
- Unauthorized access attempts

Example:

```
audit_logs (
    id INTEGER,
    user TEXT,
    action TEXT,
    resource TEXT,
    timestamp TIMESTAMP
)
```

---

# 11. Demo Checklist (Assignment Compliance)

✔ 5+ endpoints  
✔ RBAC enforcement  
✔ H3 integration  
✔ Event-driven simulation  
✔ Audit logging  
✔ Institutional constraints  
✔ Sensitive data handling  
✔ Aggregation table  

---

# 12. Conclusion

This system demonstrates:

- Institutional-scale design
- Spatially-aware access control
- Clean architecture
- Secure RBAC + ABAC enforcement
- Practical H3 integration in backend systems

Developed for INF 395 – Institutional Information Systems.