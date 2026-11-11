// Mirrors the Movie / Director structs in ../main.go
export type Director = {
  firstname: string
  lastname: string
}

export type Movie = {
  id: string
  isbn: string
  title: string
  director: Director | null
}

export type MovieInput = Omit<Movie, 'id'>
