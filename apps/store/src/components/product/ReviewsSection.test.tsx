import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import ReviewsSection from '@/components/product/ReviewsSection';
import { Review } from '@/types';

const mockReviews: Review[] = [
  {
    id: 'r1',
    author: 'Алексей К.',
    rating: 5,
    date: '2024-12-10',
    text: 'Отличный товар!',
    size: 'M',
    helpful: 12,
  },
  {
    id: 'r2',
    author: 'Мария С.',
    rating: 4,
    date: '2024-11-28',
    text: 'Хороший товар.',
    size: 'S',
    helpful: 8,
  },
  {
    id: 'r3',
    author: 'Дмитрий В.',
    rating: 5,
    date: '2024-11-15',
    text: 'Беру второй раз.',
    size: 'L',
    helpful: 5,
  },
];

describe('ReviewsSection', () => {
  it('renders average rating', () => {
    render(<ReviewsSection reviews={mockReviews} />);
    // Average = (5+4+5)/3 = 4.7 -> rounded to 5 in display, but text shows 4.7
    expect(screen.getByText('4.7')).toBeInTheDocument();
  });

  it('renders review count', () => {
    render(<ReviewsSection reviews={mockReviews} />);
    expect(screen.getByText('3 отзывов')).toBeInTheDocument();
  });

  it('renders all reviews', () => {
    render(<ReviewsSection reviews={mockReviews} />);
    expect(screen.getByText('Алексей К.')).toBeInTheDocument();
    expect(screen.getByText('Мария С.')).toBeInTheDocument();
    expect(screen.getByText('Дмитрий В.')).toBeInTheDocument();
  });

  it('renders review text', () => {
    render(<ReviewsSection reviews={mockReviews} />);
    expect(screen.getByText('Отличный товар!')).toBeInTheDocument();
  });

  it('increments helpful count on click', () => {
    render(<ReviewsSection reviews={mockReviews} />);
    const helpfulButton = screen.getAllByText(/Полезно/)[0];
    expect(helpfulButton).toHaveTextContent('Полезно (12)');
    fireEvent.click(helpfulButton);
    expect(helpfulButton).toHaveTextContent('Полезно (13)');
  });

  it('shows size badges', () => {
    render(<ReviewsSection reviews={mockReviews} />);
    expect(screen.getByText('M')).toBeInTheDocument();
    expect(screen.getByText('S')).toBeInTheDocument();
    expect(screen.getByText('L')).toBeInTheDocument();
  });

  it('handles empty reviews', () => {
    render(<ReviewsSection reviews={[]} />);
    // Average rating is "0" but there are many "0"s in the rating bars.
    // Check the review count text instead.
    expect(screen.getByText('0 отзывов')).toBeInTheDocument();
    expect(screen.getByText('Отзывы покупателей')).toBeInTheDocument();
  });
});
