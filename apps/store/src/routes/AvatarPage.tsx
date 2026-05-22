import { Suspense, useState, lazy } from "react";
import * as THREE from "three";
import { useAvatarStore } from "@/stores/avatarStore";
import { Save, RotateCcw, User } from "lucide-react";
import { motion } from "framer-motion";
import { useTranslation } from "react-i18next";
import { Helmet } from "react-helmet-async";

const AvatarModel = lazy(() => import("@/components/avatar/AvatarModel"));
const Canvas = lazy(() =>
  import("@react-three/fiber").then((mod) => ({ default: mod.Canvas })),
);
const OrbitControls = lazy(() =>
  import("@react-three/drei").then((mod) => ({ default: mod.OrbitControls })),
);

const poses = ["idle", "front", "back", "side", "tpose"];

export default function AvatarPage() {
  const { t } = useTranslation();
  const { params, setParams, setPose, currentPose } = useAvatarStore();
  const [localParams, setLocalParams] = useState(params);

  const handleSave = () => {
    setParams(localParams);
    alert("Аватар сохранен!");
  };

  const handleReset = () => {
    setLocalParams({
      gender: "male",
      height: 175,
      weight: 70,
      fatPercentage: 20,
      musclePercentage: 30,
    });
  };

  const sliders = [
    { label: t("avatar.height"), key: "height", min: 150, max: 200 },
    { label: t("avatar.weight"), key: "weight", min: 40, max: 120 },
    { label: t("avatar.fatPercentage"), key: "fatPercentage", min: 0, max: 50 },
    {
      label: t("avatar.musclePercentage"),
      key: "musclePercentage",
      min: 0,
      max: 60,
    },
  ];

  return (
    <div className="px-6 py-24 lg:px-8">
      <Helmet>
        <title>Редактор 3D-аватара — MIORU | Виртуальная примерка</title>
        <meta
          name="description"
          content="Создай своего 3D-аватара для виртуальной примерки одежды. Настрой рост, вес, процент жира и мышц для точного подбора размера."
        />
        <meta property="og:title" content="3D-аватар — MIORU" />
        <meta
          property="og:description"
          content="Создай 3D-аватара для точной виртуальной примерки одежды."
        />
        <link rel="canonical" href="https://mioru.store/avatar" />
      </Helmet>
      <div className="mx-auto max-w-6xl">
        <motion.h1
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="text-4xl font-bold tracking-tighter text-[var(--color-text-primary)] sm:text-5xl"
        >
          {t("avatar.title")}
        </motion.h1>

        <div className="mt-10 grid gap-8 lg:grid-cols-2">
          {/* 3D Preview */}
          <motion.div
            initial={{ opacity: 0, x: -30 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.6 }}
            className="rounded-2xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] overflow-hidden"
          >
            <div className="aspect-square">
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
                <Canvas camera={{ position: [0, 1.5, 3], fov: 50 }}>
                  <ambientLight intensity={0.6} />
                  <directionalLight position={[5, 5, 5]} intensity={0.8} />
                  <directionalLight position={[-3, 3, -5]} intensity={0.3} />
                  <AvatarModel
                    params={{
                      gender: localParams.gender,
                      height: localParams.height,
                      weight: localParams.weight,
                      fat: localParams.fatPercentage,
                      muscle: localParams.musclePercentage,
                    }}
                  />
                  <OrbitControls
                    enablePan={false}
                    minDistance={1.5}
                    maxDistance={6}
                    target={[0, 1, 0]}
                    autoRotate
                    autoRotateSpeed={0.5}
                  />
                  {/* Circular platform */}
                  <mesh
                    rotation={[-Math.PI / 2, 0, 0]}
                    position={[0, -0.9, 0]}
                    receiveShadow
                  >
                    <ringGeometry args={[0.35, 0.38, 64]} />
                    <meshStandardMaterial
                      color="#44944A"
                      side={THREE.DoubleSide}
                      transparent
                      opacity={0.6}
                    />
                  </mesh>
                  <mesh
                    rotation={[-Math.PI / 2, 0, 0]}
                    position={[0, -0.9, 0]}
                    receiveShadow
                  >
                    <ringGeometry args={[0.28, 0.35, 64]} />
                    <meshStandardMaterial
                      color="#44944A"
                      side={THREE.DoubleSide}
                      transparent
                      opacity={0.25}
                    />
                  </mesh>
                </Canvas>
              </Suspense>
            </div>
            <div className="border-t border-[var(--color-border-custom)] p-4">
              <p className="text-xs font-mono text-[var(--color-text-muted)] uppercase tracking-wider mb-3">
                {t("product.pose")}
              </p>
              <div className="flex gap-2">
                {poses.map((pose) => (
                  <button
                    key={pose}
                    onClick={() => setPose(pose)}
                    className={`flex-1 rounded-lg border px-3 py-2 text-xs font-medium transition-all ${
                      currentPose === pose
                        ? "border-[#44944A] bg-[#44944A] text-black"
                        : "border-[var(--color-border-custom)] text-[var(--color-text-secondary)] hover:text-white"
                    }`}
                  >
                    {pose === "tpose"
                      ? "T-pose"
                      : pose.charAt(0).toUpperCase() + pose.slice(1)}
                  </button>
                ))}
              </div>
            </div>
          </motion.div>

          {/* Controls */}
          <motion.div
            initial={{ opacity: 0, x: 30 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.6, delay: 0.1 }}
            className="space-y-6"
          >
            <div>
              <label className="text-xs font-mono text-[var(--color-text-muted)] uppercase tracking-wider">
                {t("avatar.gender")}
              </label>
              <div className="mt-2 flex gap-2">
                {(["male", "female"] as const).map((g) => (
                  <button
                    key={g}
                    onClick={() =>
                      setLocalParams({ ...localParams, gender: g })
                    }
                    className={`flex-1 rounded-xl border px-4 py-3 text-sm font-medium transition-all ${
                      localParams.gender === g
                        ? "border-[#44944A] bg-[#44944A] text-black"
                        : "border-[var(--color-border-custom)] text-[var(--color-text-secondary)] hover:text-white"
                    }`}
                  >
                    {g === "male" ? t("avatar.male") : t("avatar.female")}
                  </button>
                ))}
              </div>
            </div>

            {sliders.map((slider) => (
              <div key={slider.key}>
                <div className="flex justify-between mb-2">
                  <label className="text-sm font-medium text-[var(--color-text-primary)]">
                    {slider.label}
                  </label>
                  <span className="text-sm font-mono text-[#44944A]">
                    {localParams[slider.key as keyof typeof localParams]}
                  </span>
                </div>
                <input
                  type="range"
                  min={slider.min}
                  max={slider.max}
                  value={
                    localParams[
                      slider.key as keyof typeof localParams
                    ] as number
                  }
                  onChange={(e) =>
                    setLocalParams({
                      ...localParams,
                      [slider.key]: Number(e.target.value),
                    })
                  }
                  className="w-full h-2 rounded-lg appearance-none cursor-pointer bg-[var(--color-border-custom)] accent-[#44944A]"
                />
              </div>
            ))}

            <div className="flex gap-4 pt-4">
              <button
                onClick={handleSave}
                className="flex flex-1 items-center justify-center gap-2 rounded-xl bg-[#44944A] px-6 py-3 text-sm font-semibold text-black transition-all hover:shadow-[0_0_30px_rgba(192,254,57,0.3)]"
              >
                <Save className="h-4 w-4" />
                {t("avatar.save")}
              </button>
              <button
                onClick={handleReset}
                className="flex items-center justify-center gap-2 rounded-xl border border-[var(--color-border-custom)] px-6 py-3 text-sm text-[var(--color-text-secondary)] transition-all hover:bg-[var(--color-bg-card)] hover:text-white"
              >
                <RotateCcw className="h-4 w-4" />
                {t("avatar.reset")}
              </button>
            </div>

            <div className="rounded-xl bg-[var(--color-bg-card)] border border-[var(--color-border-custom)] p-4">
              <div className="flex items-start gap-3">
                <User className="h-5 w-5 text-[#44944A] shrink-0 mt-0.5" />
                <div>
                  <p className="text-sm text-[var(--color-text-primary)] font-medium">
                    {t("avatar.tip")}
                  </p>
                  <p className="text-xs text-[var(--color-text-secondary)] mt-1">
                    {t("avatar.tipText")}
                  </p>
                </div>
              </div>
            </div>
          </motion.div>
        </div>
      </div>
    </div>
  );
}
