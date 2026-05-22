export const COLORS = {
  bgPrimary: 'rgb(10, 10, 10)',
  bgSecondary: 'rgb(25, 25, 25)',
  textPrimary: 'rgb(255, 255, 255)',
  textSecondary: 'rgb(180, 180, 180)',
  accent: 'rgb(192, 254, 57)',
  accentSoft: 'rgb(158, 253, 98)',
  border: 'rgb(50, 50, 50)',
  textHover: 'rgb(220, 220, 220)',
} as const;

export const SIZES = ['XS', 'S', 'M', 'L', 'XL', 'XXL'] as const;
export const SHOE_SIZES = ['40', '41', '42', '43', '44', '45'] as const;

export const VIP_LEVELS = [
  { level: 0, name: 'Bronze', minXp: 0 },
  { level: 1, name: 'Silver', minXp: 1000 },
  { level: 2, name: 'Gold', minXp: 5000 },
  { level: 3, name: 'VIP', minXp: 15000 },
] as const;
