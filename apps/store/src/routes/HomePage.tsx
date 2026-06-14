import { useEffect } from "react";
import { Helmet } from "@dr.pogodin/react-helmet";
import { useCatalogStore } from "@/stores/catalogStore";
import HeroSection from "@/components/home/HeroSection";
import AnimatedStripes from "@/components/home/AnimatedStripes";
import FeaturedProducts from "@/components/home/FeaturedProducts";
import CTASection from "@/components/home/CTASection";

export default function HomePage() {
  const { fetchProducts, fetchCategories } = useCatalogStore();

  useEffect(() => {
    fetchProducts({ per_page: "6", sort: "newest" });
    fetchCategories();
  }, [fetchProducts, fetchCategories]);

  return (
    <>
      <Helmet>
        <title>MIORU — Виртуальная примерка одежды | 3D Avatar Try-On</title>
        <meta
          name="description"
          content="Создай своего 3D-аватара и примеряй одежду онлайн. Кроссовки, футболки, шорты и аксессуары с виртуальной примеркой. Точный фит, реальные пропорции."
        />
        <meta
          property="og:title"
          content="MIORU — Виртуальная примерка одежды"
        />
        <meta
          property="og:description"
          content="Создай 3D-аватара и примеряй одежду онлайн. Точный фит, реальные пропорции."
        />
        <meta property="og:type" content="website" />
        <link rel="canonical" href="https://mioru.store" />
      </Helmet>
      <HeroSection />
      <AnimatedStripes />
      <FeaturedProducts />
      <CTASection />
    </>
  );
}
