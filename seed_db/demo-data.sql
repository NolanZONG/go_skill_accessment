-- ============================================================================
-- Demo data for development / showcase.
-- Auto-loaded after 02-seed.sql via /docker-entrypoint-initdb.d (file name
-- 03-demo.sql in the container). Safe to skip in production by simply not
-- mounting this file.
--
-- Contents:
--   * 4 classes (Grade 9-12) with 3 sections (A, B, C)
--   * 4 departments (Science, Math, English, PE)
--   * 3 leave policies (Annual, Sick, Casual)
--   * 2 notice recipient types (Teacher -> department, Student -> class)
--   * 4 teachers (role_id = 2) with profiles + department + class assignments
--   * 8 students (role_id = 3) with profiles + class/section + parents info
--   * 4 notices (2 approved, 1 pending, 1 draft) authored by admin
--   * 3 sample leave requests (1 approved, 2 pending)
--
-- All references are looked up by stable natural keys (email, name) instead of
-- hard-coded ids, so this script does not depend on the auto-increment state
-- left behind by 02-seed.sql.
-- ============================================================================

-- ---------------------------------------------------------------------------
-- Academic structure
-- ---------------------------------------------------------------------------
INSERT INTO classes (name, sections) VALUES
    ('Grade 9',  'A,B,C'),
    ('Grade 10', 'A,B,C'),
    ('Grade 11', 'A,B'),
    ('Grade 12', 'A,B')
ON CONFLICT (name) DO NOTHING;

INSERT INTO sections (name) VALUES
    ('A'),
    ('B'),
    ('C')
ON CONFLICT (name) DO NOTHING;

INSERT INTO departments (name) VALUES
    ('Science'),
    ('Mathematics'),
    ('English'),
    ('Physical Education')
ON CONFLICT (name) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Leave policies
-- ---------------------------------------------------------------------------
INSERT INTO leave_policies (name) VALUES
    ('Annual Leave'),
    ('Sick Leave'),
    ('Casual Leave');

-- ---------------------------------------------------------------------------
-- Notice recipient types (used by the Add Notice form to filter audience)
-- primary_dependent_select is a literal SQL string executed by the backend
-- (see frontend/src/domains/notice/...). Keep it to read-only SELECTs only.
-- ---------------------------------------------------------------------------
INSERT INTO notice_recipient_types (role_id, primary_dependent_name, primary_dependent_select)
VALUES
    (2, 'department', 'SELECT name FROM departments'),
    (3, 'class',      'SELECT name FROM classes');

-- ---------------------------------------------------------------------------
-- Teachers (role_id = 2)
-- Passwords intentionally NULL: these are demo seed accounts that show up in
-- lists but cannot log in. Set is_active + is_email_verified so the UI does
-- not flag them as "pending invite".
-- ---------------------------------------------------------------------------
INSERT INTO users (name, email, role_id, is_active, is_email_verified, created_dt) VALUES
    ('Alice Johnson', 'alice.j@school.local', 2, true, true, now()),
    ('Brian Smith',   'brian.s@school.local', 2, true, true, now()),
    ('Carla Lopez',   'carla.l@school.local', 2, true, true, now()),
    ('David Chen',    'david.c@school.local', 2, true, true, now())
ON CONFLICT (email) DO NOTHING;

INSERT INTO user_profiles (
    user_id, gender, marital_status, phone, dob, join_dt,
    qualification, experience, department_id,
    current_address, permanent_address,
    father_name, mother_name, emergency_phone
)
SELECT u.id,
       v.gender, v.marital_status, v.phone, v.dob::DATE, v.join_dt::DATE,
       v.qualification, v.experience,
       (SELECT id FROM departments WHERE name = v.dept_name),
       v.address, v.address,
       v.father, v.mother, v.emergency
FROM (VALUES
    ('alice.j@school.local', 'Female', 'Married', '+1-202-555-0101', '1985-04-12', '2020-08-01', 'MSc Physics',     '8 years',  'Science',            '12 Maple St', 'Anna',  'Mary',  '+1-202-555-0901'),
    ('brian.s@school.local', 'Male',   'Single',  '+1-202-555-0102', '1990-09-21', '2021-09-01', 'MA English',      '5 years',  'English',            '7 Oak Ave',   'Roger', 'Eve',   '+1-202-555-0902'),
    ('carla.l@school.local', 'Female', 'Single',  '+1-202-555-0103', '1988-02-04', '2019-08-15', 'MSc Mathematics', '6 years',  'Mathematics',        '21 Pine Rd',  'Luis',  'Rosa',  '+1-202-555-0903'),
    ('david.c@school.local', 'Male',   'Married', '+1-202-555-0104', '1982-11-30', '2018-08-15', 'MEd PE',          '10 years', 'Physical Education', '5 Elm Ln',    'Wei',   'Lin',   '+1-202-555-0904')
) AS v(email, gender, marital_status, phone, dob, join_dt, qualification, experience, dept_name, address, father, mother, emergency)
JOIN users u ON u.email = v.email
ON CONFLICT (user_id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Class teacher assignments (also used by student_add_update to resolve
-- reporter_id automatically when adding a student via the UI).
-- ---------------------------------------------------------------------------
INSERT INTO class_teachers (teacher_id, class_name, section_name)
SELECT u.id, ct.class_name, ct.section_name
FROM (VALUES
    ('alice.j@school.local', 'Grade 10', 'A'),
    ('brian.s@school.local', 'Grade 10', 'B'),
    ('carla.l@school.local', 'Grade 11', 'A'),
    ('david.c@school.local', 'Grade 9',  'A')
) AS ct(email, class_name, section_name)
JOIN users u ON u.email = ct.email;

-- ---------------------------------------------------------------------------
-- Students (role_id = 3)
-- reporter_id is wired to the matching class teacher when available.
-- ---------------------------------------------------------------------------
INSERT INTO users (name, email, role_id, is_active, is_email_verified, reporter_id, created_dt)
SELECT v.name, v.email, 3, true, true,
       (SELECT id FROM users WHERE email = v.reporter_email),
       now()
FROM (VALUES
    ('Emma Wang',    'emma.w@school.local',    'alice.j@school.local'),
    ('Frank Miller', 'frank.m@school.local',   'alice.j@school.local'),
    ('Grace Lee',    'grace.l@school.local',   'brian.s@school.local'),
    ('Henry Park',   'henry.p@school.local',   'brian.s@school.local'),
    ('Iris Kim',     'iris.k@school.local',    'carla.l@school.local'),
    ('Jack Brown',   'jack.b@school.local',    'david.c@school.local'),
    ('Karen Davis',  'karen.d@school.local',   'david.c@school.local'),
    ('Liam Garcia',  'liam.g@school.local',    'carla.l@school.local')
) AS v(name, email, reporter_email)
ON CONFLICT (email) DO NOTHING;

INSERT INTO user_profiles (
    user_id, gender, phone, dob,
    class_name, section_name, roll,
    admission_dt,
    father_name, father_phone, mother_name, mother_phone,
    current_address, permanent_address
)
SELECT u.id,
       v.gender, v.phone, v.dob::DATE,
       v.class_name, v.section_name, v.roll,
       v.admission_dt::DATE,
       v.father_name, v.father_phone, v.mother_name, v.mother_phone,
       v.address, v.address
FROM (VALUES
    ('emma.w@school.local',  'Female', '+1-202-555-0201', '2008-03-15', 'Grade 10', 'A', 1, '2023-08-01', 'Wei Wang',     '+1-202-555-1001', 'Lin Wang',     '+1-202-555-1101', '101 Cedar St'),
    ('frank.m@school.local', 'Male',   '+1-202-555-0202', '2008-07-22', 'Grade 10', 'A', 2, '2023-08-01', 'James Miller', '+1-202-555-1002', 'Anna Miller',  '+1-202-555-1102', '202 Birch St'),
    ('grace.l@school.local', 'Female', '+1-202-555-0203', '2008-11-10', 'Grade 10', 'B', 1, '2023-08-01', 'Min Lee',      '+1-202-555-1003', 'Sun Lee',      '+1-202-555-1103', '303 Walnut Ave'),
    ('henry.p@school.local', 'Male',   '+1-202-555-0204', '2008-01-05', 'Grade 10', 'B', 2, '2023-08-01', 'Joon Park',    '+1-202-555-1004', 'Hye Park',     '+1-202-555-1104', '404 Spruce Rd'),
    ('iris.k@school.local',  'Female', '+1-202-555-0205', '2007-05-19', 'Grade 11', 'A', 1, '2022-08-01', 'Dae Kim',      '+1-202-555-1005', 'Mi Kim',       '+1-202-555-1105', '505 Aspen Ln'),
    ('jack.b@school.local',  'Male',   '+1-202-555-0206', '2009-09-08', 'Grade 9',  'A', 1, '2024-08-01', 'Tom Brown',    '+1-202-555-1006', 'Sue Brown',    '+1-202-555-1106', '606 Poplar Ct'),
    ('karen.d@school.local', 'Female', '+1-202-555-0207', '2009-12-25', 'Grade 9',  'A', 2, '2024-08-01', 'Mark Davis',   '+1-202-555-1007', 'Jane Davis',   '+1-202-555-1107', '707 Willow Way'),
    ('liam.g@school.local',  'Male',   '+1-202-555-0208', '2007-08-30', 'Grade 11', 'A', 2, '2022-08-01', 'Carlos Garcia','+1-202-555-1008', 'Maria Garcia', '+1-202-555-1108', '808 Hickory Dr')
) AS v(email, gender, phone, dob, class_name, section_name, roll, admission_dt, father_name, father_phone, mother_name, mother_phone, address)
JOIN users u ON u.email = v.email
ON CONFLICT (user_id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Notices (authored by admin)
-- notice_status: 1=Draft, 2=Submit for Review, 5=Approve (see 02-seed.sql)
-- ---------------------------------------------------------------------------
INSERT INTO notices (
    title, description, status,
    recipient_type, recipient_role_id, recipient_first_field,
    author_id, reviewer_id, reviewed_dt, created_dt
)
SELECT v.title, v.description, v.status,
       v.recipient_type, v.recipient_role_id, v.first_field,
       (SELECT id FROM users WHERE email = 'admin@school-admin.com'),
       CASE WHEN v.status = 5 THEN (SELECT id FROM users WHERE email = 'admin@school-admin.com') END,
       CASE WHEN v.status = 5 THEN now() END,
       now()
FROM (VALUES
    ('Welcome to the new academic year', 'School starts on August 1st. Please collect your timetable from the front office.', 5, 'EV', NULL::INTEGER, NULL::TEXT),
    ('Library closure for renovation',   'The school library will be closed from June 10-15 for renovation work.',             5, 'EV', NULL::INTEGER, NULL::TEXT),
    ('Parent-teacher meeting',           'Grade 10 parent-teacher meeting scheduled for Saturday at 10 AM.',                    2, 'SP', 3,             'Grade 10'),
    ('Staff retreat',                    'Annual staff retreat planning meeting on Friday after school.',                      1, 'SP', 2,             NULL)
) AS v(title, description, status, recipient_type, recipient_role_id, first_field);

-- ---------------------------------------------------------------------------
-- Leave policy assignments (give every teacher every policy)
-- ---------------------------------------------------------------------------
INSERT INTO user_leave_policy (user_id, leave_policy_id)
SELECT u.id, lp.id
FROM users u
CROSS JOIN leave_policies lp
WHERE u.role_id = 2
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- Sample leave requests
-- leave_status: 1=On Review, 2=Approved, 3=Cancelled (see 02-seed.sql)
-- ---------------------------------------------------------------------------
INSERT INTO user_leaves (
    user_id, leave_policy_id,
    from_dt, to_dt, note,
    submitted_dt, status,
    approver_id, approved_dt
)
SELECT u.id,
       (SELECT id FROM leave_policies WHERE name = v.policy_name),
       v.from_dt::DATE, v.to_dt::DATE, v.note,
       now(), v.status,
       CASE WHEN v.status = 2 THEN (SELECT id FROM users WHERE email = 'admin@school-admin.com') END,
       CASE WHEN v.status = 2 THEN now() END
FROM (VALUES
    ('alice.j@school.local', 'Sick Leave',   '2026-06-01', '2026-06-02', 'Flu',             1),
    ('brian.s@school.local', 'Casual Leave', '2026-06-10', '2026-06-11', 'Family event',    2),
    ('carla.l@school.local', 'Annual Leave', '2026-07-15', '2026-07-20', 'Summer vacation', 1)
) AS v(email, policy_name, from_dt, to_dt, note, status)
JOIN users u ON u.email = v.email;
