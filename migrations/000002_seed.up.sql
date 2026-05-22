-- Seed restaurant users (password: "restaurant123" hashed with bcrypt)
INSERT INTO users (id, email, password_hash, name, role) VALUES
    ('a1111111-1111-1111-1111-111111111111', 'mario@restaurant.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Mario', 'restaurant'),
    ('a2222222-2222-2222-2222-222222222222', 'sakura@restaurant.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Sakura', 'restaurant'),
    ('a3333333-3333-3333-3333-333333333333', 'ali@restaurant.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Ali', 'restaurant');

INSERT INTO restaurants (id, user_id, name, description, address) VALUES
    ('b1111111-1111-1111-1111-111111111111', 'a1111111-1111-1111-1111-111111111111', 'Mario''s Italian', 'Authentic Italian cuisine with fresh pasta and wood-fired pizzas.', '123 Main St, New York, NY'),
    ('b2222222-2222-2222-2222-222222222222', 'a2222222-2222-2222-2222-222222222222', 'Sakura Sushi', 'Traditional Japanese sushi and ramen.', '456 Oak Ave, San Francisco, CA'),
    ('b3333333-3333-3333-3333-333333333333', 'a3333333-3333-3333-3333-333333333333', 'Ali''s Kebab House', 'Middle Eastern grills and fresh mezze platters.', '789 Pine Rd, Chicago, IL');

INSERT INTO menu_items (restaurant_id, name, description, price, available) VALUES
    -- Mario's Italian
    ('b1111111-1111-1111-1111-111111111111', 'Margherita Pizza', 'Classic tomato, mozzarella, and basil.', 12.99, true),
    ('b1111111-1111-1111-1111-111111111111', 'Spaghetti Carbonara', 'Creamy pasta with pancetta and parmesan.', 14.50, true),
    ('b1111111-1111-1111-1111-111111111111', 'Tiramisu', 'Traditional Italian coffee dessert.', 8.00, true),
    ('b1111111-1111-1111-1111-111111111111', 'Bruschetta', 'Grilled bread with tomato and basil topping.', 7.50, false),
    -- Sakura Sushi
    ('b2222222-2222-2222-2222-222222222222', 'Salmon Nigiri (6pc)', 'Fresh Atlantic salmon on seasoned rice.', 11.00, true),
    ('b2222222-2222-2222-2222-222222222222', 'Tonkotsu Ramen', 'Rich pork bone broth with chashu and egg.', 15.00, true),
    ('b2222222-2222-2222-2222-222222222222', 'Edamame', 'Steamed soybeans with sea salt.', 5.00, true),
    ('b2222222-2222-2222-2222-222222222222', 'Dragon Roll', 'Eel, avocado, and cucumber roll.', 16.50, true),
    -- Ali's Kebab House
    ('b3333333-3333-3333-3333-333333333333', 'Chicken Shawarma Plate', 'Marinated chicken with rice, salad, and garlic sauce.', 13.00, true),
    ('b3333333-3333-3333-3333-333333333333', 'Lamb Kebab', 'Grilled lamb skewers with hummus and pita.', 16.00, true),
    ('b3333333-3333-3333-3333-333333333333', 'Falafel Wrap', 'Crispy falafel with tahini sauce in fresh pita.', 10.50, true),
    ('b3333333-3333-3333-3333-333333333333', 'Baklava', 'Layered pastry with honey and pistachios.', 6.00, true);
