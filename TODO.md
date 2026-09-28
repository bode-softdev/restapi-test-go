Instruksi Pengerjaan
1.	Buat REST API untuk manajemen produk
Endpoint utama:
/api/products
Struktur data produk minimal:
{
"id": 1,
"title": "Awesome T-Shirt", "price": 99.99,
"description": "High-quality cotton t-shirt", "category": "Clothes",
"images": ["https://placeimg.com/640/480/any"],
“created_at”: “2025-01-01 15:01:04”,
“created_by”: “Jhon Doe”, “created_by_id”: “1”,
“updated_at”: “2025-01-01 15:01:04”,
“updated_by”: “Jhon Doe”, “updated_by_id”: “1”,
}


2.	Buat REST API untuk authentication
Endpoint utama:
/api/auth
Struktur data auth register minimal:
{
"username": “jhon_doe”,
"password": "supersecret", "password_confirmation": "supersecret",
}
 
3.	Fitur yang harus dibuat
A.	GET /api/products
●	Mengembalikan semua produk.
●	Dapat difilter dengan query:
o	?search=keyword → mencari berdasarkan nama produk
o	?category=Clothes → filter berdasarkan kategori
o	?limit=10&page=2 → pagination
●	Implementasi Cache basic (Optional)


B.	GET /api/products/:id
●	Mengembalikan detail satu produk.
●	Jika tidak ditemukan, kembalikan status 404 dengan pesan error JSON.
●	Implementasi Cache basic (Optional)


C.	POST /api/products
●	Menambahkan produk baru.
●	Validasi wajib: title, price, category, dan images (minimal 1).
●	Jika validasi gagal, kembalikan status 400 dengan pesan error.
●	Jika berhasil, kembalikan produk baru yang disimpan.
●	Pastikan endpoint hanya bisa diakses setelah proses authorization valid.
●	Pastikan endpoint tidak dapat di hit lebih dari 1x dalam 5 detik. (Optional)

D.	PUT /api/products/:id
●	Mengedit data produk berdasarkan ID.
●	Jika produk tidak ditemukan, kembalikan 404.
●	Pastikan endpoint hanya bisa diakses setelah proses authorization valid.
●	Pastikan endpoint tidak dapat di hit lebih dari 1x dalam 5 detik. (Optional)

E.	DELETE /api/products/:id
●	Menghapus produk berdasarkan ID.
●	Jika produk tidak ditemukan, kembalikan 404.
●	Pastikan endpoint hanya bisa diakses setelah proses authorization valid.
●	Pastikan endpoint tidak dapat di hit lebih dari 1x dalam 5 detik. (Optional)
 
F.	POST /api/auth/register
●	Menambahkan data register baru.
●	Pastikan endpoint tidak dapat di hit lebih dari 3x dalam 60 detik. (Optional)

G.	POST /api/auth/login
●	Login menggunakan username dan password
●	Mengembalikan authentication_token dan refresh_token
●	Pastikan endpoint tidak dapat di hit lebih dari 3x dalam 60 detik. (Optional)

4.	Delivery
●	Gunakan bahasa Go (Golang)
●	Pastikan aplikasi dapat dijalankan menggunakan docker.
●	Pastikan aplikasi web dari berbagai domain untuk bisa berinteraksi dengan API yang dibuat.
●	API Dokumentasi (Swagger/Postman)
●	URL Public Remote Repository
