import { useState, lazy, Suspense } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { User, ImageIcon, ChevronLeft, ChevronRight } from "lucide-react";
import type { Product } from "@/types";
import { useAvatarStore } from "@/stores/avatarStore";
import { useTranslation } from "react-i18next";

const AvatarModel = lazy(() => import("@/components/avatar/AvatarModel"));
const Canvas = lazy(() =>
  import("@react-three/fiber").then((mod) => ({ default: mod.Canvas })),
);
const OrbitControls = lazy(() =>
  import("@react-three/drei").then((mod) => ({ default: mod.OrbitControls })),
);

interface ProductGalleryProps {
  product: Product;
}

const poses = ["idle", "front", "back", "side", "tpose"];

export default function ProductGallery({ product }: ProductGalleryProps) {
  const { t } = useTranslation();
  const [viewMode, setViewMode] = useState<"avatar" | "photos">("avatar");
  const [selectedPhotoIndex, setSelectedPhotoIndex] = useState(0);
  const [failedImages, setFailedImages] = useState<Set<number>>(new Set());
  const { params, currentPose, setPose } = useAvatarStore();

  const imageUrls = product.images.map((img) => img.url);

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
      {/* View Mode Toggle */}
      <div className="absolute top-4 left-1/2 -translate-x-1/2 z-20 flex bg-[var(--color-bg-secondary)] border border-[var(--color-border-custom)] rounded-full p-1">
        <button
          onClick={() => setViewMode("avatar")}
          className={`flex items-center gap-2 px-4 py-2 rounded-full text-xs font-medium transition-all min-h-[44px] ${
            viewMode === "avatar"
              ? "bg-[#44944A] text-black"
              : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
          }`}
        >
          <User className="h-3.5 w-3.5" />
          {t("product.view3D")}
        </button>
        <button
          onClick={() => setViewMode("photos")}
          className={`flex items-center gap-2 px-4 py-2 rounded-full text-xs font-medium transition-all min-h-[44px] ${
            viewMode === "photos"
              ? "bg-[#44944A] text-black"
              : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
          }`}
        >
          <ImageIcon className="h-3.5 w-3.5" />
          {t("product.viewPhotos")}
        </button>
      </div>

      {/* Main Display */}
      <div className="relative aspect-square overflow-hidden rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)]">
        <AnimatePresence mode="wait">
          {viewMode === "avatar" ? (
            <motion.div
              key="avatar"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.3 }}
              className="absolute inset-0"
              style={{ touchAction: "pan-y" }}
            >
              <Suspense
                fallback={
                  <div className="h-full flex items-center justify-center">
                    <div className="text-center">
                      <div className="text-6xl mb-2">👤</div>
                      <p className="text-[var(--color-text-muted)]">
                        {t("avatar.loading")}
                      </p>
                    </div>
                  </div>
                }
              >
                <Canvas
                  camera={{ position: [0, 1.5, 3], fov: 50 }}
                  dpr={[1, 2]}
                >
                  <ambientLight intensity={0.8} />
                  <pointLight position={[10, 10, 10]} />
                  <AvatarModel
                    params={{
                      gender: params.gender,
                      height: params.height,
                      weight: params.weight,
                      fat: params.fatPercentage,
                      muscle: params.musclePercentage,
                    }}
                  />
                  <OrbitControls
                    enablePan={false}
                    minDistance={2}
                    maxDistance={5}
                    target={[0, 1, 0]}
                  />
                  <gridHelper
                    args={[5, 20, "#333", "#222"]}
                    position={[0, 0, 0]}
                  />
                </Canvas>
              </Suspense>

              {/* Mobile hint */}
              <div className="absolute top-4 right-4 md:hidden bg-[var(--color-bg-primary)]/60 backdrop-blur-sm border border-[var(--color-border-custom)] rounded-full px-3 py-1 text-[10px] text-[var(--color-text-muted)]">
                Двойной тап — вращение
              </div>

              {/* Product Badge on Avatar */}
              <div className="absolute bottom-4 left-4 right-4">
                <div className="bg-[var(--color-bg-primary)]/80 backdrop-blur-sm border border-[var(--color-border-custom)] rounded-xl p-3">
                  <p className="text-xs font-mono text-[#44944A] uppercase tracking-wider">
                    3D Примерка
                  </p>
                  <p className="text-sm text-[var(--color-text-primary)] mt-1 font-medium">
                    {product.name}
                  </p>
                  <p className="text-xs text-[var(--color-text-secondary)] mt-0.5">
                    {t("product.pose")}:{" "}
                    {currentPose === "tpose"
                      ? "T-pose"
                      : currentPose.charAt(0).toUpperCase() +
                        currentPose.slice(1)}
                  </p>
                </div>
              </div>
            </motion.div>
          ) : (
            <motion.div
              key="photos"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
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
                    {imageUrls[selectedPhotoIndex] || "Нет фото"}
                  </p>
                </div>
              )}
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {/* Thumbnails / Pose Selector */}
      <div className="mt-4">
        {viewMode === "avatar" ? (
          <div className="flex gap-2">
            {poses.map((pose) => (
              <button
                key={pose}
                onClick={() => setPose(pose)}
                className={`flex-1 rounded-lg border px-3 py-2.5 text-xs font-medium transition-all min-h-[44px] ${
                  currentPose === pose
                    ? "border-[#44944A] bg-[#44944A] text-black"
                    : "border-[var(--color-border-custom)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:border-[var(--color-text-muted)]"
                }`}
              >
                {pose === "tpose"
                  ? "T-pose"
                  : pose.charAt(0).toUpperCase() + pose.slice(1)}
              </button>
            ))}
          </div>
        ) : (
          <div className="flex gap-2 overflow-x-auto pb-2">
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
    </div>
  );
}
