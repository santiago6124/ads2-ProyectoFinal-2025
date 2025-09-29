// MongoDB initialization script for Portfolio API

// Switch to the portfolio database
db = db.getSiblingDB('portfolio_db');

// Create application user
db.createUser({
  user: 'portfolio_user',
  pwd: 'portfolio_pass123',
  roles: [
    {
      role: 'readWrite',
      db: 'portfolio_db'
    }
  ]
});

// Create collections with validation schemas
db.createCollection('portfolios', {
  validator: {
    $jsonSchema: {
      bsonType: 'object',
      required: ['user_id', 'total_value', 'total_invested', 'holdings'],
      properties: {
        _id: {
          bsonType: 'objectId'
        },
        user_id: {
          bsonType: 'long',
          minimum: 1,
          description: 'User ID must be a positive integer'
        },
        total_value: {
          bsonType: 'decimal',
          minimum: 0,
          description: 'Total value must be a non-negative decimal'
        },
        total_invested: {
          bsonType: 'decimal',
          minimum: 0,
          description: 'Total invested must be a non-negative decimal'
        },
        total_cash: {
          bsonType: 'decimal',
          minimum: 0,
          description: 'Total cash must be a non-negative decimal'
        },
        holdings: {
          bsonType: 'array',
          items: {
            bsonType: 'object',
            required: ['symbol', 'quantity', 'average_cost', 'current_price'],
            properties: {
              symbol: {
                bsonType: 'string',
                minLength: 1,
                maxLength: 20,
                description: 'Symbol must be a non-empty string'
              },
              quantity: {
                bsonType: 'decimal',
                minimum: 0,
                description: 'Quantity must be non-negative'
              },
              average_cost: {
                bsonType: 'decimal',
                minimum: 0,
                description: 'Average cost must be non-negative'
              },
              current_price: {
                bsonType: 'decimal',
                minimum: 0,
                description: 'Current price must be non-negative'
              }
            }
          }
        },
        created_at: {
          bsonType: 'date',
          description: 'Created at must be a date'
        },
        updated_at: {
          bsonType: 'date',
          description: 'Updated at must be a date'
        }
      }
    }
  }
});

db.createCollection('portfolio_snapshots', {
  validator: {
    $jsonSchema: {
      bsonType: 'object',
      required: ['portfolio_id', 'user_id', 'timestamp', 'interval', 'value'],
      properties: {
        _id: {
          bsonType: 'objectId'
        },
        portfolio_id: {
          bsonType: 'objectId',
          description: 'Portfolio ID must be an ObjectId'
        },
        user_id: {
          bsonType: 'long',
          minimum: 1,
          description: 'User ID must be a positive integer'
        },
        timestamp: {
          bsonType: 'date',
          description: 'Timestamp must be a date'
        },
        interval: {
          bsonType: 'string',
          enum: ['hourly', 'daily', 'weekly', 'monthly', 'manual'],
          description: 'Interval must be one of the allowed values'
        },
        value: {
          bsonType: 'object',
          required: ['total', 'invested', 'cash', 'profit_loss'],
          properties: {
            total: {
              bsonType: 'decimal',
              minimum: 0
            },
            invested: {
              bsonType: 'decimal',
              minimum: 0
            },
            cash: {
              bsonType: 'decimal',
              minimum: 0
            },
            profit_loss: {
              bsonType: 'decimal'
            }
          }
        }
      }
    }
  }
});

// Create indexes for optimal performance
print('Creating indexes for portfolios collection...');

// Portfolios indexes
db.portfolios.createIndex({ 'user_id': 1 }, { unique: true, name: 'idx_user_id' });
db.portfolios.createIndex({ 'updated_at': -1 }, { name: 'idx_updated_at' });
db.portfolios.createIndex({ 'metadata.needs_recalculation': 1 }, { name: 'idx_needs_recalc' });
db.portfolios.createIndex({ 'metadata.last_calculated': -1 }, { name: 'idx_last_calculated' });
db.portfolios.createIndex({ 'total_value': -1 }, { name: 'idx_total_value' });
db.portfolios.createIndex({ 'profit_loss_percentage': -1 }, { name: 'idx_profit_loss_pct' });
db.portfolios.createIndex({ 'performance.daily_change_percentage': -1 }, { name: 'idx_daily_change' });
db.portfolios.createIndex({ 'performance.weekly_change_percentage': -1 }, { name: 'idx_weekly_change' });
db.portfolios.createIndex({ 'performance.monthly_change_percentage': -1 }, { name: 'idx_monthly_change' });
db.portfolios.createIndex({ 'performance.yearly_change_percentage': -1 }, { name: 'idx_yearly_change' });

// Holdings indexes (for aggregation queries)
db.portfolios.createIndex({ 'holdings.symbol': 1 }, { name: 'idx_holdings_symbol' });
db.portfolios.createIndex({ 'holdings.category': 1 }, { name: 'idx_holdings_category' });

print('Creating indexes for portfolio_snapshots collection...');

// Portfolio snapshots indexes
db.portfolio_snapshots.createIndex({ 'user_id': 1, 'timestamp': -1 }, { name: 'idx_user_timestamp' });
db.portfolio_snapshots.createIndex({ 'portfolio_id': 1, 'interval': 1, 'timestamp': -1 }, { name: 'idx_portfolio_interval_timestamp' });
db.portfolio_snapshots.createIndex({ 'timestamp': -1 }, { expireAfterSeconds: 7776000, name: 'idx_timestamp_ttl' }); // 90 days TTL
db.portfolio_snapshots.createIndex({ 'interval': 1, 'timestamp': -1 }, { name: 'idx_interval_timestamp' });
db.portfolio_snapshots.createIndex({ 'tags': 1 }, { sparse: true, name: 'idx_tags' });
db.portfolio_snapshots.createIndex({ 'user_id': 1, 'interval': 1 }, { name: 'idx_user_interval' });

// Compound indexes for common queries
db.portfolio_snapshots.createIndex({
  'user_id': 1,
  'timestamp': -1,
  'interval': 1
}, { name: 'idx_user_timestamp_interval' });

// Create sample data for development/testing
print('Creating sample data...');

// Sample portfolio
const samplePortfolioId = ObjectId();
const sampleUserId = NumberLong(1001);

db.portfolios.insertOne({
  _id: samplePortfolioId,
  user_id: sampleUserId,
  total_value: NumberDecimal('10000.00'),
  total_invested: NumberDecimal('9500.00'),
  total_cash: NumberDecimal('500.00'),
  profit_loss: NumberDecimal('500.00'),
  profit_loss_percentage: NumberDecimal('0.0526'),
  holdings: [
    {
      _id: ObjectId(),
      symbol: 'BTC',
      name: 'Bitcoin',
      quantity: NumberDecimal('0.5'),
      average_cost: NumberDecimal('45000.00'),
      current_price: NumberDecimal('50000.00'),
      current_value: NumberDecimal('25000.00'),
      profit_loss: NumberDecimal('2500.00'),
      profit_loss_percentage: NumberDecimal('0.1111'),
      percentage_of_portfolio: NumberDecimal('0.25'),
      category: 'cryptocurrency',
      created_at: new Date(),
      updated_at: new Date()
    },
    {
      _id: ObjectId(),
      symbol: 'ETH',
      name: 'Ethereum',
      quantity: NumberDecimal('10'),
      average_cost: NumberDecimal('2000.00'),
      current_price: NumberDecimal('2200.00'),
      current_value: NumberDecimal('22000.00'),
      profit_loss: NumberDecimal('2000.00'),
      profit_loss_percentage: NumberDecimal('0.10'),
      percentage_of_portfolio: NumberDecimal('0.22'),
      category: 'cryptocurrency',
      created_at: new Date(),
      updated_at: new Date()
    }
  ],
  performance: {
    daily_change: NumberDecimal('100.00'),
    daily_change_percentage: NumberDecimal('0.01'),
    weekly_change: NumberDecimal('500.00'),
    weekly_change_percentage: NumberDecimal('0.05'),
    monthly_change: NumberDecimal('1000.00'),
    monthly_change_percentage: NumberDecimal('0.10'),
    yearly_change: NumberDecimal('2000.00'),
    yearly_change_percentage: NumberDecimal('0.25')
  },
  risk_metrics: {
    volatility_30d: NumberDecimal('0.25'),
    sharpe_ratio: NumberDecimal('1.2'),
    sortino_ratio: NumberDecimal('1.5'),
    max_drawdown: NumberDecimal('0.15'),
    var_95: NumberDecimal('0.05'),
    cvar_95: NumberDecimal('0.08'),
    beta: NumberDecimal('1.1'),
    alpha: NumberDecimal('0.03')
  },
  diversification: {
    holdings_count: 2,
    effective_holdings: NumberDecimal('1.8'),
    concentration_index: NumberDecimal('0.32'),
    largest_position_percentage: NumberDecimal('0.25'),
    sector_count: 1,
    sector_diversification_ratio: NumberDecimal('0.5')
  },
  metadata: {
    last_calculated: new Date(),
    needs_recalculation: false,
    calculation_version: '1.0'
  },
  created_at: new Date(),
  updated_at: new Date()
});

// Sample snapshots
const baseDate = new Date();
for (let i = 0; i < 30; i++) {
  const snapshotDate = new Date(baseDate.getTime() - (i * 24 * 60 * 60 * 1000)); // i days ago
  const baseValue = 9500 + (Math.random() * 1000 - 500); // Random value around 9500-10500

  db.portfolio_snapshots.insertOne({
    _id: ObjectId(),
    portfolio_id: samplePortfolioId,
    user_id: sampleUserId,
    timestamp: snapshotDate,
    interval: 'daily',
    value: {
      total: NumberDecimal(baseValue.toFixed(2)),
      invested: NumberDecimal('9500.00'),
      cash: NumberDecimal('500.00'),
      profit_loss: NumberDecimal((baseValue - 9500).toFixed(2)),
      profit_loss_percentage: NumberDecimal(((baseValue - 9500) / 9500).toFixed(4))
    },
    holdings_snapshot: [
      {
        symbol: 'BTC',
        name: 'Bitcoin',
        quantity: NumberDecimal('0.5'),
        price: NumberDecimal((48000 + Math.random() * 4000).toFixed(2)),
        value: NumberDecimal((24000 + Math.random() * 2000).toFixed(2)),
        percentage: NumberDecimal('0.25'),
        category: 'cryptocurrency'
      },
      {
        symbol: 'ETH',
        name: 'Ethereum',
        quantity: NumberDecimal('10'),
        price: NumberDecimal((2100 + Math.random() * 200).toFixed(2)),
        value: NumberDecimal((21000 + Math.random() * 2000).toFixed(2)),
        percentage: NumberDecimal('0.22'),
        category: 'cryptocurrency'
      }
    ],
    metrics: {
      volatility: NumberDecimal((0.2 + Math.random() * 0.1).toFixed(4)),
      sharpe_ratio: NumberDecimal((1.0 + Math.random() * 0.5).toFixed(4)),
      holdings_count: 2
    },
    created_at: snapshotDate
  });
}

print('MongoDB initialization completed successfully!');
print('Database: portfolio_db');
print('Collections created: portfolios, portfolio_snapshots');
print('Sample data inserted: 1 portfolio, 30 daily snapshots');
print('Indexes created for optimal performance');
print('Application user created: portfolio_user');