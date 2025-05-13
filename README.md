# CampusClaim-
A lost and found app for uni students

cmd/server: Entry point for your Go app

internal/: Contains the core logic, private to your app (idiomatic Go)

models/: Split into postgres/ and mongo/ for clarity

api/v1/: For RESTful HTTP route handlers

chat/: Handles WebSocket server + MongoDB integration

search/: Might contain FAISS, feature extraction, or external API hooks

storage/: For handling uploaded images and saved file paths

migrations/: Stores .sql files (e.g., using golang-migrate)