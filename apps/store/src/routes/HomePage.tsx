import { Helmet } from "react-helmet-async";
import HeroSection from "@/components/home/HeroSection";
import HorizontalCategories from "@/components/home/HorizontalCategories";
import AnimatedStripes from "@/components/home/AnimatedStripes";
import FeaturedProducts from "@/components/home/FeaturedProducts";
import CTASection from "@/components/home/CTASection";

export default function HomePage() {
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
      <HorizontalCategories />
      <AnimatedStripes />
      <FeaturedProducts />
      <CTASection />
    </>
  );
}
