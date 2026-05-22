import { useState } from 'react';
import { motion } from 'framer-motion';
import { Star, ThumbsUp, User } from 'lucide-react';
import { Review } from '@/types';

interface ReviewsSectionProps {
  reviews: Review[];
}

function StarRating({ rating }: { rating: number }) {
  return (
    <div className="flex gap-0.5">
      {[1, 2, 3, 4, 5].map((star) => (
        <Star
          key={star}
          className={`h-4 w-4 ${
            star <= rating
              ? 'fill-[#44944A] text-[#44944A]'
              : 'fill-transparent text-[var(--color-border-light)]'
          }`}
        />
      ))}
    </div>
  );
}

function RatingBar({ label, count, total }: { label: string; count: number; total: number }) {
  const percentage = total > 0 ? (count / total) * 100 : 0;
  return (
    <div className="flex items-center gap-3">
      <span className="text-xs text-[var(--color-text-secondary)] w-8">{label}</span>
      <div className="flex-1 h-2 rounded-full bg-[var(--color-border-custom)] overflow-hidden">
        <motion.div
          initial={{ width: 0 }}
          animate={{ width: `${percentage}%` }}
          transition={{ duration: 0.8, delay: 0.2 }}
          className="h-full rounded-full bg-[#44944A]"
        />
      </div>
      <span className="text-xs text-[var(--color-text-muted)] w-6 text-right">{count}</span>
    </div>
  );
}

export default function ReviewsSection({ reviews }: ReviewsSectionProps) {
  const [helpfulReviews, setHelpfulReviews] = useState<Set<string>>(new Set());

  const averageRating =
    reviews.length > 0
      ? (reviews.reduce((sum, r) => sum + r.rating, 0) / reviews.length).toFixed(1)
      : '0';

  const ratingCounts = [5, 4, 3, 2, 1].map((star) => ({
    star,
    count: reviews.filter((r) => r.rating === star).length,
  }));

  const toggleHelpful = (id: string) => {
    setHelpfulReviews((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  return (
    <div className="mt-12">
      <h3 className="text-lg font-semibold text-[var(--color-text-primary)] mb-6">Отзывы покупателей</h3>

      <div className="grid gap-8 lg:grid-cols-3">
        {/* Rating Summary */}
        <div className="lg:col-span-1">
          <div className="p-6 rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)]">
            <div className="flex items-end gap-3 mb-6">
              <span className="text-5xl font-bold text-[var(--color-text-primary)]">{averageRating}</span>
              <div className="mb-1.5">
                <StarRating rating={Math.round(Number(averageRating))} />
                <p className="text-xs text-[var(--color-text-secondary)] mt-1">{reviews.length} отзывов</p>
              </div>
            </div>

            <div className="space-y-2">
              {ratingCounts.map(({ star, count }) => (
                <RatingBar
                  key={star}
                  label={`${star}★`}
                  count={count}
                  total={reviews.length}
                />
              ))}
            </div>
          </div>
        </div>

        {/* Reviews List */}
        <div className="lg:col-span-2 space-y-4">
          {reviews.map((review, idx) => (
            <motion.div
              key={review.id}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.4, delay: idx * 0.1 }}
              className="p-5 rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)]"
            >
              <div className="flex items-start justify-between gap-4">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-full bg-[var(--color-border-custom)] flex items-center justify-center">
                    <User className="h-5 w-5 text-[var(--color-text-secondary)]" />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-[var(--color-text-primary)]">{review.author}</p>
                    <div className="flex items-center gap-2 mt-0.5">
                      <StarRating rating={review.rating} />
                      <span className="text-xs text-[var(--color-text-muted)]">•</span>
                      <span className="text-xs text-[var(--color-text-muted)]">
                        {new Date(review.date).toLocaleDateString('ru-RU')}
                      </span>
                    </div>
                  </div>
                </div>
                <span className="text-xs font-mono text-[#44944A] bg-[#44944A]/10 px-2 py-1 rounded-full">
                  {review.size}
                </span>
              </div>

              <p className="mt-4 text-sm text-[var(--color-text-secondary)] leading-relaxed">{review.text}</p>

              <button
                onClick={() => toggleHelpful(review.id)}
                className={`mt-4 inline-flex items-center gap-1.5 text-xs transition-colors ${
                  helpfulReviews.has(review.id)
                    ? 'text-[#44944A]'
                    : 'text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]'
                }`}
              >
                <ThumbsUp className="h-3.5 w-3.5" />
                Полезно ({review.helpful + (helpfulReviews.has(review.id) ? 1 : 0)})
              </button>
            </motion.div>
          ))}
        </div>
      </div>
    </div>
  );
}
