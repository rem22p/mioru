import { api } from './api';

export interface AdminUser {
	id: number;
	username: string;
	first_name: string;
	last_name: string;
	email: string;
	display_name: string;
	avatar_color: string;
	role: string;
	created_at: string;
}

export const fetchUsers = () => api<AdminUser[]>('/api/admin/users');

export const createUser = (body: {
	first_name: string;
	last_name: string;
	email: string;
	username: string;
	password: string;
}) =>
	api<{ username: string; email: string; display_name: string; role: string }>(
		'/api/auth/register',
		{
			method: 'POST',
			body: JSON.stringify(body),
		},
	);
