# Backend - Student Management System API

A robust Node.js REST API built with Express.js and PostgreSQL for managing school operations with comprehensive authentication, authorization, and CRUD operations.

## 🚀 Quick Start

### Prerequisites
- Node.js (v16 or higher)
- PostgreSQL (v12 or higher)
- npm or yarn

### Installation & Setup
```bash
# Install dependencies
npm install

# Set up environment variables
cp .env.example .env
# Edit .env with your configuration

# Set up database
createdb school_mgmt
psql -d school_mgmt -f ../seed_db/tables.sql
psql -d school_mgmt -f ../seed_db/seed-db.sql

# Start the server
npm start
```

### Environment Configuration
Create a `.env` file with the following variables:
```env
PORT=5007
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/school_mgmt
JWT_ACCESS_TOKEN_SECRET=your_access_token_secret
JWT_REFRESH_TOKEN_SECRET=your_refresh_token_secret
CSRF_TOKEN_SECRET=your_csrf_secret
JWT_ACCESS_TOKEN_TIME_IN_MS=900000
JWT_REFRESH_TOKEN_TIME_IN_MS=28800000
CSRF_TOKEN_TIME_IN_MS=950000
MAIL_FROM_USER=your-email@domain.com
EMAIL_VERIFICATION_TOKEN_SECRET=your_email_verification_secret
EMAIL_VERIFICATION_TOKEN_TIME_IN_MS=18000000
PASSWORD_SETUP_TOKEN_TIME_IN_MS=300000
PASSWORD_SETUP_TOKEN_SECRET=your_password_setup_secret
UI_URL=http://localhost:5173
API_URL=http://localhost:5007
COOKIE_DOMAIN=localhost
RESEND_API_KEY=your_resend_api_key
```

## 🛠️ Technology Stack

### Core Technologies
- **Node.js** - JavaScript runtime
- **Express.js** - Web application framework
- **PostgreSQL** - Primary database
- **JWT** - Authentication tokens
- **Argon2** - Password hashing

### Key Dependencies
- **express-async-handler** - Async error handling
- **cors** - Cross-origin resource sharing
- **cookie-parser** - Cookie parsing middleware
- **dotenv** - Environment variable management
- **pg** - PostgreSQL client
- **uuid** - UUID generation
- **zod** - Runtime type validation
- **resend** - Email service

## 📁 Project Structure

```
src/
├── config/                 # Configuration files
│   ├── database.js        # Database connection setup
│   └── env.js             # Environment variables
├── middlewares/           # Express middlewares
│   ├── auth.js            # Authentication middleware
│   ├── csrf.js            # CSRF protection
│   ├── error-handler.js   # Global error handling
│   └── validation.js      # Request validation
├── modules/               # Feature-based API modules
│   ├── auth/              # Authentication endpoints
│   │   ├── auth-controller.js
│   │   ├── auth-service.js
│   │   ├── auth-router.js
│   │   └── auth-repository.js
│   ├── students/          # Student management
│   │   ├── students-controller.js
│   │   ├── students-service.js
│   │   ├── students-router.js
│   │   └── students-repository.js
│   ├── notices/           # Notice management
│   ├── leave/             # Leave management
│   ├── staff/             # Staff management
│   └── departments/       # Department management
├── routes/                # API route definitions
│   ├── v1.js              # Version 1 API routes
│   └── index.js           # Route aggregation
├── shared/                # Shared utilities and repositories
│   ├── repository/        # Common database operations
│   ├── errors/            # Custom error classes
│   └── validators/        # Shared validation schemas
├── templates/             # Email templates
│   ├── password-setup.html
│   └── email-verification.html
├── utils/                 # Utility functions
│   ├── jwt-handle.js      # JWT operations
│   ├── csrf-handle.js     # CSRF token handling
│   ├── email-service.js   # Email sending utilities
│   └── helpers.js         # General helper functions
├── app.js                 # Express app configuration
└── server.js              # Server entry point
```

## 🔐 Authentication & Security

### JWT Authentication
- **Access Tokens**: Short-lived tokens (15 minutes) for API access
- **Refresh Tokens**: Long-lived tokens (8 hours) for token renewal
- **Token Rotation**: Automatic token refresh mechanism

### Security Features
- **CSRF Protection**: HMAC-based CSRF tokens
- **Password Hashing**: Argon2 for secure password storage
- **Role-Based Access Control**: Granular permissions system
- **Request Validation**: Zod schema validation
- **Secure Cookies**: HttpOnly, Secure, SameSite cookies

### Authentication Flow
```
1. User login → Validate credentials
2. Generate access + refresh tokens
3. Set secure HTTP-only cookies
4. Client includes tokens in requests
5. Middleware validates tokens
6. Automatic token refresh when needed
```

## 📚 API Documentation

### Base URL
```
http://localhost:5007/api/v1
```

### Authentication Endpoints

#### POST /auth/login
Login user and get authentication tokens.
```json
{
  "email": "admin@school-admin.com",
  "password": "3OU4zn3q6Zh9"
}
```

#### POST /auth/logout
Logout user and invalidate tokens.
```json
{
  "message": "Logged out successfully"
}
```

#### GET /auth/refresh
Refresh access token using refresh token.
```json
{
  "accessToken": "new_access_token",
  "user": { "id": 1, "name": "Admin", "role": "admin" }
}
```

### Student Management Endpoints

#### GET /students
Get all students with pagination and filtering.
```
Query Parameters:
- page: Page number (default: 1)
- limit: Items per page (default: 10)
- search: Search term
- class: Filter by class
- section: Filter by section
```

#### POST /students
Create a new student.
```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "class_name": "Grade 10",
  "section_name": "A",
  "roll": 101,
  "dob": "2005-01-15",
  "father_name": "Robert Doe",
  "father_phone": "+1234567890"
}
```

#### PUT /students/:id
Update student information.
```json
{
  "name": "John Smith",
  "phone": "+1234567891"
}
```

#### DELETE /students/:id
Delete a student record.
```json
{
  "message": "Student deleted successfully"
}
```

### Notice Management Endpoints

#### GET /notices
Get all notices with filtering.
```
Query Parameters:
- status: Filter by status (draft, published, archived)
- author_id: Filter by author
- recipient_type: Filter by recipient type
```

#### POST /notices
Create a new notice.
```json
{
  "title": "Important Announcement",
  "description": "This is an important notice for all students.",
  "recipient_type": "all",
  "recipient_role_id": null
}
```

#### PUT /notices/:id
Update notice.
```json
{
  "title": "Updated Announcement",
  "description": "Updated notice content"
}
```

#### DELETE /notices/:id
Delete a notice.

### Leave Management Endpoints

#### GET /leave/requests
Get leave requests with filtering.

#### POST /leave/requests
Submit a new leave request.
```json
{
  "from_dt": "2024-01-15",
  "to_dt": "2024-01-17",
  "note": "Family emergency",
  "leave_policy_id": 1
}
```

#### PUT /leave/requests/:id/approve
Approve a leave request.

#### PUT /leave/requests/:id/reject
Reject a leave request.

### Staff Management Endpoints

#### GET /staffs
Get all staff members.

#### POST /staffs
Add new staff member.
```json
{
  "name": "Jane Teacher",
  "email": "jane@school.com",
  "role_id": 2,
  "department_id": 1,
  "join_dt": "2024-01-01"
}
```

## 🗄️ Database Schema

### Key Tables
- **users**: User accounts and basic information
- **user_profiles**: Extended user profile data
- **roles**: System roles and permissions
- **classes**: Academic classes
- **sections**: Class sections
- **departments**: Organizational departments
- **notices**: System notices and announcements
- **user_leaves**: Leave requests and approvals
- **access_controls**: Permission definitions
- **permissions**: Role-permission mappings

### Relationships
- Users belong to roles
- Users have profiles
- Students belong to classes and sections
- Staff belong to departments
- Notices have authors and recipients
- Leave requests belong to users

## 🔧 Development Guidelines

### Code Structure
```javascript
// Controller pattern
const handleGetStudents = asyncHandler(async (req, res) => {
  const { page = 1, limit = 10, search } = req.query;
  const result = await studentService.getStudents({ page, limit, search });
  res.json(result);
});

// Service pattern
const getStudents = async ({ page, limit, search }) => {
  const offset = (page - 1) * limit;
  return await studentRepository.findStudents({ offset, limit, search });
};

// Repository pattern
const findStudents = async ({ offset, limit, search }) => {
  const query = `
    SELECT * FROM users u
    JOIN user_profiles up ON u.id = up.user_id
    WHERE u.role_id = $1
    ${search ? 'AND u.name ILIKE $2' : ''}
    LIMIT $3 OFFSET $4
  `;
  // Execute query and return results
};
```

### Error Handling
```javascript
// Custom error classes
class ApiError extends Error {
  constructor(statusCode, message) {
    super(message);
    this.statusCode = statusCode;
  }
}

// Global error handler
const errorHandler = (err, req, res, next) => {
  const statusCode = err.statusCode || 500;
  res.status(statusCode).json({
    success: false,
    message: err.message,
    ...(process.env.NODE_ENV === 'development' && { stack: err.stack })
  });
};
```

### Validation
```javascript
// Zod schema validation
const createStudentSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  email: z.string().email('Invalid email'),
  class_name: z.string().optional(),
  section_name: z.string().optional(),
  roll: z.number().int().positive().optional()
});

// Middleware usage
const validateCreateStudent = (req, res, next) => {
  try {
    createStudentSchema.parse(req.body);
    next();
  } catch (error) {
    throw new ApiError(400, 'Validation failed');
  }
};
```

## 🧪 Testing

### Running Tests
```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Generate coverage report
npm run test:coverage
```

### Test Structure
```javascript
// Example test
describe('Student Controller', () => {
  describe('GET /students', () => {
    it('should return paginated students', async () => {
      const response = await request(app)
        .get('/api/v1/students')
        .set('Authorization', `Bearer ${token}`)
        .expect(200);
      
      expect(response.body.data).toBeInstanceOf(Array);
      expect(response.body.pagination).toBeDefined();
    });
  });
});
```

## 🚀 Deployment

### Production Build
```bash
# Install production dependencies
npm ci --only=production

# Start production server
NODE_ENV=production npm start
```

### Docker Deployment
```dockerfile
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
EXPOSE 5007
CMD ["npm", "start"]
```

### Environment Setup
- Set up PostgreSQL database
- Configure environment variables
- Set up SSL certificates for HTTPS
- Configure reverse proxy (Nginx)
- Set up monitoring and logging

## 🐛 Known Issues & Solutions

### Issue 1: Student CRUD Operations Incomplete
**Problem**: Some CRUD operations for students are missing or incomplete.
**Location**: `/src/modules/students/students-controller.js`
**Solution**: 
- Implement missing endpoints (CREATE, UPDATE, DELETE)
- Add proper validation and error handling
- Test all operations thoroughly

### Issue 2: Notice Description Not Saving
**Problem**: Notice description field not being saved properly.
**Location**: `/src/modules/notices/notices-service.js`
**Solution**: 
- Check database query parameters
- Verify request body parsing
- Add proper validation for description field

## 📊 Performance Considerations

### Database Optimization
- Use connection pooling
- Implement proper indexing
- Use prepared statements
- Optimize complex queries

### Caching Strategy
- Implement Redis for session storage
- Cache frequently accessed data
- Use ETags for conditional requests

### Security Best Practices
- Regular security audits
- Keep dependencies updated
- Implement rate limiting
- Use HTTPS in production

## 📄 Scripts Reference

| Script | Description |
|--------|-------------|
| `npm start` | Start production server |
| `npm run dev` | Start development server with nodemon |
| `npm test` | Run test suite |
| `npm run lint` | Run ESLint |
| `npm run format` | Format code with Prettier |

---

For frontend documentation, see [../frontend/README.md](../frontend/README.md)
