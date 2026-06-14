import { useEffect, useState, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { fetchUsers, createUser, deleteUser, type AdminUser } from "@/lib/users";
import { Plus, User, Mail, Shield, X, Trash2 } from "lucide-react";

export default function Users() {
	const [users, setUsers] = useState<AdminUser[]>([]);
	const [loading, setLoading] = useState(true);
	const [formOpen, setFormOpen] = useState(false);
	const [form, setForm] = useState({
		username: "",
		first_name: "",
		last_name: "",
		email: "",
		password: "",
	});
	const [formError, setFormError] = useState("");
	const [submitting, setSubmitting] = useState(false);
	const [deleteTarget, setDeleteTarget] = useState<AdminUser | null>(null);

	const loadUsers = useCallback(async () => {
		setLoading(true);
		try {
			const data = await fetchUsers();
			setUsers(data);
		} catch {
			// handled
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		loadUsers();
	}, [loadUsers]);

	const handleCreate = async () => {
		setFormError("");
		if (!form.username.trim() || !form.password || !form.email.trim() || !form.first_name.trim() || !form.last_name.trim()) {
			setFormError("Все поля обязательны");
			return;
		}
		if (form.password.length < 8) {
			setFormError("Пароль минимум 8 символов");
			return;
		}
		setSubmitting(true);
		try {
			await createUser(form);
			setFormOpen(false);
			setForm({ username: "", first_name: "", last_name: "", email: "", password: "" });
			loadUsers();
		} catch (err) {
			setFormError(err instanceof Error ? err.message : "Ошибка");
		} finally {
			setSubmitting(false);
		}
	};

	const handleDelete = async () => {
		if (!deleteTarget) return;
		try {
			await deleteUser(deleteTarget.username);
			setDeleteTarget(null);
			loadUsers();
		} catch {
			// handled
		}
	};

	const updateForm = (field: string, value: string) =>
		setForm((f) => ({ ...f, [field]: value }));

	return (
		<div className="p-6 lg:p-8">
			{/* Header */}
			<motion.div
				initial={{ opacity: 0, y: 20 }}
				animate={{ opacity: 1, y: 0 }}
				transition={{ duration: 0.6, ease: [0.16, 1, 0.3, 1] }}
				className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-8"
			>
				<div>
					<h1 className="text-2xl lg:text-3xl font-bold tracking-tighter text-[var(--color-text-primary)]">
						Пользователи
					</h1>
					<p className="text-sm text-[var(--color-text-secondary)] mt-1">
						{users.length} {users.length === 1 ? "пользователь" : users.length < 5 ? "пользователя" : "пользователей"}
					</p>
				</div>
				<button
					onClick={() => setFormOpen(true)}
					data-testid="user-add"
					className="flex items-center gap-2 rounded-xl bg-[#44944A] px-5 py-2.5 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(68,148,74,0.3)]"
				>
					<Plus className="h-4 w-4" />
					Создать пользователя
				</button>
			</motion.div>

			{/* Table */}
			<motion.div
				initial={{ opacity: 0, y: 20 }}
				animate={{ opacity: 1, y: 0 }}
				transition={{ duration: 0.6, delay: 0.1, ease: [0.16, 1, 0.3, 1] }}
				className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] overflow-hidden"
			>
				{loading ? (
					<div className="flex items-center justify-center py-20">
						<div className="h-6 w-6 animate-spin rounded-full border-2 border-[var(--color-border-custom)] border-t-[#44944A]" />
					</div>
				) : users.length === 0 ? (
					<div className="flex flex-col items-center justify-center py-20 text-[var(--color-text-muted)]">
						<User className="h-10 w-10 mb-3 opacity-30" />
						<p className="text-sm">Нет пользователей</p>
					</div>
				) : (
					<table className="w-full">
						<thead>
							<tr className="border-b border-[var(--color-border-custom)]">
								<th className="text-left px-6 py-3.5 text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
									Пользователь
								</th>
								<th className="text-left px-6 py-3.5 text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
									Email
								</th>
								<th className="text-left px-6 py-3.5 text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
									Роль
								</th>
								<th className="text-left px-6 py-3.5 text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
									Создан
								</th>
								<th className="text-right px-6 py-3.5 text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">
									Действия
								</th>
							</tr>
						</thead>
						<tbody>
							{users.map((u) => (
								<tr
									key={u.id}
									data-testid="user-row"
									data-username={u.username}
									className="border-b border-[var(--color-border-custom)] last:border-0 hover:bg-[var(--color-bg-primary)]/50 transition-colors"
								>
									<td className="px-6 py-4">
										<div className="flex items-center gap-3">
											<div
												className="h-9 w-9 rounded-full flex items-center justify-center text-sm font-bold text-white flex-shrink-0"
												style={{ backgroundColor: u.avatar_color || "#44944A" }}
											>
												{(u.display_name || u.username)[0]?.toUpperCase() || "?"}
											</div>
											<div>
												<div className="text-sm font-medium text-[var(--color-text-primary)]">
													{u.display_name || `${u.first_name} ${u.last_name}`.trim()}
												</div>
												<div className="text-xs text-[var(--color-text-muted)]">
													@{u.username}
												</div>
											</div>
										</div>
									</td>
									<td className="px-6 py-4 text-sm text-[var(--color-text-secondary)]">
										{u.email}
									</td>
									<td className="px-6 py-4">
										<span
											className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium ${
												u.role === "admin"
													? "bg-[#44944A]/10 text-[#44944A]"
													: "bg-[var(--color-bg-secondary)] text-[var(--color-text-muted)]"
											}`}
										>
											<Shield className="h-3 w-3" />
											{u.role}
										</span>
									</td>
									<td className="px-6 py-4 text-sm text-[var(--color-text-muted)]">
										{u.created_at
											? new Date(u.created_at).toLocaleDateString("ru-RU")
											: "—"}
									</td>
									<td className="px-6 py-4 text-right">
										<button
											onClick={() => setDeleteTarget(u)}
											data-testid="user-delete"
											className="inline-flex items-center justify-center h-8 w-8 rounded-lg text-[var(--color-text-muted)] hover:text-red-500 hover:bg-red-500/10 transition-colors"
											title="Удалить"
										>
											<Trash2 className="h-4 w-4" />
										</button>
									</td>
								</tr>
							))}
						</tbody>
					</table>
				)}
			</motion.div>

			{/* Create User Modal */}
			<AnimatePresence>
				{formOpen && (
					<motion.div
						initial={{ opacity: 0 }}
						animate={{ opacity: 1 }}
						exit={{ opacity: 0 }}
						className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4"
						onClick={() => setFormOpen(false)}
					>
						<motion.div
							initial={{ opacity: 0, scale: 0.95, y: 20 }}
							animate={{ opacity: 1, scale: 1, y: 0 }}
							exit={{ opacity: 0, scale: 0.95, y: 20 }}
							transition={{ duration: 0.3, ease: [0.16, 1, 0.3, 1] }}
							onClick={(e) => e.stopPropagation()}
							className="w-full max-w-md rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-6 shadow-2xl"
						>
							{/* Modal header */}
							<div className="flex items-center justify-between mb-6">
								<div>
									<h2 className="text-lg font-bold text-[var(--color-text-primary)]">
										Новый пользователь
									</h2>
									<p className="text-xs text-[var(--color-text-muted)] mt-0.5">
										Создать аккаунт администратора
									</p>
								</div>
								<button
									onClick={() => setFormOpen(false)}
									className="h-8 w-8 flex items-center justify-center rounded-lg text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-secondary)] transition-colors"
								>
									<X className="h-4 w-4" />
								</button>
							</div>

							{/* Form */}
							<div className="space-y-4">
								<div className="grid grid-cols-2 gap-3">
									<div>
										<label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1.5">
											Имя
										</label>
										<input
											type="text"
											placeholder="Иван"
											data-testid="uf-first-name"
											value={form.first_name}
											onChange={(e) => updateForm("first_name", e.target.value)}
											className="w-full rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-3 py-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] transition-colors"
										/>
									</div>
									<div>
										<label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1.5">
											Фамилия
										</label>
										<input
											type="text"
											placeholder="Иванов"
											data-testid="uf-last-name"
											value={form.last_name}
											onChange={(e) => updateForm("last_name", e.target.value)}
											className="w-full rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-3 py-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] transition-colors"
										/>
									</div>
								</div>

								<div>
									<label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1.5">
										Никнейм
									</label>
									<div className="relative">
										<User className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--color-text-muted)]" />
										<input
											type="text"
											placeholder="admin"
											data-testid="uf-username"
											value={form.username}
											onChange={(e) => updateForm("username", e.target.value)}
											className="w-full rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] pl-9 pr-3 py-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] transition-colors"
										/>
									</div>
								</div>

								<div>
									<label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1.5">
										Email
									</label>
									<div className="relative">
										<Mail className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--color-text-muted)]" />
										<input
											type="email"
											placeholder="admin@mioru.store"
											data-testid="uf-email"
											value={form.email}
											onChange={(e) => updateForm("email", e.target.value)}
											className="w-full rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] pl-9 pr-3 py-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] transition-colors"
										/>
									</div>
								</div>

								<div>
									<label className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1.5">
										Пароль
									</label>
									<input
										type="password"
										placeholder="Минимум 8 символов"
										data-testid="uf-password"
										value={form.password}
										onChange={(e) => updateForm("password", e.target.value)}
										className="w-full rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-3 py-2 text-sm text-[var(--color-text-primary)] outline-none focus:border-[#44944A] placeholder:text-[var(--color-text-muted)] transition-colors"
									/>
								</div>

								{formError && (
									<p className="text-sm text-red-500 px-1">{formError}</p>
								)}

								<button
									onClick={handleCreate}
									disabled={submitting}
									data-testid="uf-submit"
									className="w-full rounded-xl bg-[#44944A] px-4 py-2.5 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(68,148,74,0.3)] disabled:opacity-50 disabled:cursor-not-allowed"
								>
									{submitting ? "Создание..." : "Создать"}
								</button>
							</div>
						</motion.div>
					</motion.div>
				)}
			</AnimatePresence>

			{/* Delete Confirmation Modal */}
			<AnimatePresence>
				{deleteTarget && (
					<motion.div
						initial={{ opacity: 0 }}
						animate={{ opacity: 1 }}
						exit={{ opacity: 0 }}
						className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4"
						onClick={() => setDeleteTarget(null)}
					>
						<motion.div
							initial={{ opacity: 0, scale: 0.95, y: 20 }}
							animate={{ opacity: 1, scale: 1, y: 0 }}
							exit={{ opacity: 0, scale: 0.95, y: 20 }}
							transition={{ duration: 0.3, ease: [0.16, 1, 0.3, 1] }}
							onClick={(e) => e.stopPropagation()}
							className="w-full max-w-sm rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-6 shadow-2xl"
						>
							<h2 className="text-lg font-bold text-[var(--color-text-primary)] mb-2">
								Удалить пользователя?
							</h2>
							<p className="text-sm text-[var(--color-text-secondary)] mb-6">
								Пользователь <strong>@{deleteTarget.username}</strong> будет удалён безвозвратно.
							</p>
							<div className="flex gap-3">
								<button
									onClick={() => setDeleteTarget(null)}
									className="flex-1 rounded-xl bg-[var(--color-bg-primary)] border border-[var(--color-border-custom)] px-4 py-2.5 text-sm font-medium text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] transition-colors"
								>
									Отмена
								</button>
								<button
									onClick={handleDelete}
									data-testid="user-delete-confirm"
									className="flex-1 rounded-xl bg-red-500 px-4 py-2.5 text-sm font-semibold text-white transition-all hover:bg-red-600"
								>
									Удалить
								</button>
							</div>
						</motion.div>
					</motion.div>
				)}
			</AnimatePresence>
		</div>
	);
}
