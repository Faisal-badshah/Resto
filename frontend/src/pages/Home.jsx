import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { fetchRestaurant, postOrder, postSubscribe } from '../api';

export default function Home({ restaurantId: defaultId }) {
  const { id } = useParams();
  const restaurantId = id || defaultId;
  
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [cart, setCart] = useState([]);
  
  useEffect(() => {
    fetchRestaurant(restaurantId)
      .then(res => {
        setData(res);
        setLoading(false);
      })
      .catch(err => {
        console.error(err);
        setLoading(false);
      });
  }, [restaurantId]);
  
  const addToCart = (item) => {
    setCart([...cart, item]);
  };
  
  const submitOrder = async () => {
    if (cart.length === 0) return alert('Cart is empty');
    
    const total = cart.reduce((sum, item) => sum + item.price, 0);
    const customerName = prompt('Your name:');
    const customerPhone = prompt('Your phone:');
    const customerAddress = prompt('Your address:');
    
    if (!customerName || !customerPhone) return;
    
    try {
      await postOrder(restaurantId, {
        items: cart,
        total,
        customerName,
        customerPhone,
        customerAddress,
        customerEmail: '',
        notes: ''
      });
      alert('Order placed successfully!');
      setCart([]);
    } catch (err) {
      alert('Failed to place order');
    }
  };
  
  if (loading) return <div style={{ padding: 20 }}>Loading...</div>;
  if (!data) return <div style={{ padding: 20 }}>Restaurant not found</div>;
  
  return (
    <div style={{ padding: 20, maxWidth: 1200, margin: '0 auto' }}>
      <header style={{ marginBottom: 40 }}>
        <h1>{data.restaurant.name}</h1>
        <p>{data.restaurant.story}</p>
        <div>
          <strong>Address:</strong> {data.restaurant.address}<br/>
          <strong>Phone:</strong> {data.restaurant.phone}<br/>
          <strong>Hours:</strong> {data.restaurant.hours}
        </div>
      </header>
      
      <section style={{ marginBottom: 40 }}>
        <h2>Menu</h2>
        {data.menus.map(category => (
          <div key={category.category} style={{ marginBottom: 30 }}>
            <h3>{category.category}</h3>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(250px, 1fr))', gap: 20 }}>
              {category.items.map((item, idx) => (
                <div key={idx} style={{ border: '1px solid #ddd', padding: 15, borderRadius: 8 }}>
                  {item.img && <img src={item.img} alt={item.name} style={{ width: '100%', height: 150, objectFit: 'cover', borderRadius: 4 }} />}
                  <h4>{item.name}</h4>
                  <p>{item.desc}</p>
                  <p><strong>${item.price.toFixed(2)}</strong></p>
                  {item.available && (
                    <button onClick={() => addToCart(item)} style={{ padding: '8px 16px', cursor: 'pointer' }}>
                      Add to Cart
                    </button>
                  )}
                  {!item.available && <span style={{ color: '#999' }}>Not available</span>}
                </div>
              ))}
            </div>
          </div>
        ))}
      </section>
      
      {cart.length > 0 && (
        <section style={{ position: 'fixed', bottom: 0, left: 0, right: 0, background: '#fff', borderTop: '2px solid #333', padding: 20 }}>
          <div style={{ maxWidth: 1200, margin: '0 auto', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div>
              <strong>Cart ({cart.length} items)</strong>
              <span style={{ marginLeft: 20 }}>
                Total: ${cart.reduce((sum, item) => sum + item.price, 0).toFixed(2)}
              </span>
            </div>
            <button onClick={submitOrder} style={{ padding: '10px 20px', fontSize: 16, cursor: 'pointer' }}>
              Place Order
            </button>
          </div>
        </section>
      )}
      
      {data.galleries.images.length > 0 && (
        <section style={{ marginBottom: 40 }}>
          <h2>Gallery</h2>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: 15 }}>
            {data.galleries.images.map((img, idx) => (
              <div key={idx}>
                <img src={img} alt="" style={{ width: '100%', height: 200, objectFit: 'cover', borderRadius: 4 }} />
                {data.galleries.captions[idx] && <p style={{ marginTop: 5, fontSize: 14 }}>{data.galleries.captions[idx]}</p>}
              </div>
            ))}
          </div>
        </section>
      )}
      
      {data.reviews.length > 0 && (
        <section style={{ marginBottom: 40 }}>
          <h2>Reviews</h2>
          {data.reviews.map((review, idx) => (
            <div key={idx} style={{ borderBottom: '1px solid #eee', padding: '15px 0' }}>
              <div><strong>{review.name}</strong> - {'⭐'.repeat(review.rating)}</div>
              <p>{review.comment}</p>
              <small>{new Date(review.date).toLocaleDateString()}</small>
            </div>
          ))}
        </section>
      )}
    </div>
  );
}
