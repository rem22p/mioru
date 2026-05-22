import { useParams, useNavigate } from 'react-router-dom';
import { products } from '@/lib/data';
import ProductPageClient from './ProductPageClient';

export default function ProductPage() {
  const { slug } = useParams<{ slug: string }>();
  const navigate = useNavigate();
  const product = products.find((p) => p.slug === slug);

  if (!product) {
    // Redirect to catalog if product not found
    navigate('/catalog', { replace: true });
    return null;
  }

  return <ProductPageClient product={product} />;
}
