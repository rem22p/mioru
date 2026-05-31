-- 005_super_admin: promote the initial admin to super_admin.
-- super_admin has unrestricted access (users CRUD), while
-- regular admin has product/category access only.

UPDATE users SET role = 'super_admin' WHERE username = 'rem22p';
