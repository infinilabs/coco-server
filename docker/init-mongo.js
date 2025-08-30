// MongoDB initialization script for testing  
db = db.getSiblingDB('coco_test');  
  
// Create test user  
db.createUser({  
  user: 'coco_test',  
  pwd: 'test_password',  
  roles: [  
    {  
      role: 'readWrite',  
      db: 'coco_test'  
    }  
  ]  
});  
  
// Create test collections with sample data  
db.articles.insertMany([  
  {  
    title: "Sample Article 1",  
    content: "This is sample content for testing",  
    category: "Technology",  
    tags: ["mongodb", "database"],  
    url: "https://example.com/article1",  
    updated_at: new Date(),  
    status: "published"  
  },  
  {  
    title: "Sample Article 2",   
    content: "Another sample content for testing",  
    category: "Programming",  
    tags: ["go", "backend"],  
    url: "https://example.com/article2",  
    updated_at: new Date(),  
    status: "draft"  
  }  
]);  
  
// Create indexes for better performance  
db.articles.createIndex({ "updated_at": 1 });  
db.articles.createIndex({ "status": 1 });  
db.articles.createIndex({ "category": 1 });