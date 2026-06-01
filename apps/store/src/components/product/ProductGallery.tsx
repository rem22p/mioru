import { useState } from "react";
import { motion } from "framer-motion";
import { ChevronLeft, ChevronRight } from "lucide-react";
import type { Product } from "@/types";
import { getImageUrl } from "@/lib/api";

interface ProductGalleryProps {
  product: Product;
}

export default function ProductGallery({ product }: ProductGalleryProps) {
  const [selectedPhotoIndex, setSelectedPhotoIndex] = useState(0);
  const [failedImages, setFailedImages] = useState<Set<number>>(new Set());

  const imageUrls = product.images.map((img) => getImageUrl(img.url));

  const handleImageError = (index: number) => {
    setFailedImages((prev) => new Set(prev).add(index));
  };

  const nextPhoto = () => {
    setSelectedPhotoIndex((prev) => (prev + 1) % imageUrls.length);
  };

  const prevPhoto = () => {
    setSelectedPhotoIndex(
      (prev) => (prev - 1 + imageUrls.length) % imageUrls.length,
    );
  };

  return (
    <div className="relative">
      {/* Main Photo Display */}
      <div className="relative aspect-[4/5] overflow-hidden rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)]">
        <motion.div
          key={selectedPhotoIndex}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.3 }}
          className="absolute inset-0 flex items-center justify-center"
        >
          {imageUrls.length > 0 && !failedImages.has(selectedPhotoIndex) ? (
            <div className="relative w-full h-full">
              <img
                src={imageUrls[selectedPhotoIndex]}
                alt={`${product.name} — фото ${selectedPhotoIndex + 1}`}
                className="object-cover w-full h-full"
                onError={() => handleImageError(selectedPhotoIndex)}
              />

              {/* Navigation Arrows */}
              {imageUrls.length > 1 && (
                <>
                  <button
                    onClick={prevPhoto}
                    className="absolute left-3 top-1/2 -translate-y-1/2 w-11 h-11 rounded-full bg-[var(--color-bg-primary)]/60 border border-[var(--color-border-custom)] flex items-center justify-center text-[var(--color-text-primary)] hover:bg-[var(--color-bg-primary)] hover:border-[#44944A] transition-all"
                    aria-label="Предыдущее фото"
                  >
                    <ChevronLeft className="h-5 w-5" />
                  </button>
                  <button
                    onClick={nextPhoto}
                    className="absolute right-3 top-1/2 -translate-y-1/2 w-11 h-11 rounded-full bg-[var(--color-bg-primary)]/60 border border-[var(--color-border-custom)] flex items-center justify-center text-[var(--color-text-primary)] hover:bg-[var(--color-bg-primary)] hover:border-[#44944A] transition-all"
                    aria-label="Следующее фото"
                  >
                    <ChevronRight className="h-5 w-5" />
                  </button>
                  <div className="absolute top-4 right-4 bg-[var(--color-bg-primary)]/60 backdrop-blur-sm border border-[var(--color-border-custom)] rounded-full px-3 py-1 text-xs font-mono text-[var(--color-text-primary)]">
                    {selectedPhotoIndex + 1} / {imageUrls.length}
                  </div>
                </>
              )}
            </div>
          ) : (
            <div className="text-center flex flex-col items-center justify-center">
              <span className="text-6xl">📦</span>
              <p className="mt-4 text-xs font-mono text-[var(--color-text-muted)] uppercase tracking-wider">
                Нет фото
              </p>
            </div>
          )}
        </motion.div>
      </div>

      {/* Thumbnails */}
      {imageUrls.length > 0 && (
        <div className="mt-4 flex gap-2 overflow-x-auto pb-2">
          {imageUrls.map((img, idx) => (
            <button
              key={img}
              onClick={() => setSelectedPhotoIndex(idx)}
              className={`relative flex-shrink-0 w-20 h-20 rounded-xl overflow-hidden border transition-all ${
                selectedPhotoIndex === idx
                  ? "border-[#44944A] ring-2 ring-[#44944A]/20"
                  : "border-[var(--color-border-custom)] hover:border-[var(--color-text-muted)]"
              }`}
            >
              {!failedImages.has(idx) ? (
                <img
                  src={img}
                  alt={`${product.name} thumbnail ${idx + 1}`}
                  className="object-cover w-full h-full"
                  onError={() => handleImageError(idx)}
                />
              ) : (
                <div className="absolute inset-0 flex items-center justify-center text-2xl bg-[var(--color-bg-card)]">
                  <span className="text-2xl">📦</span>
                </div>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
