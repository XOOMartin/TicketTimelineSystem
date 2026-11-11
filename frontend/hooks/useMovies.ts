import { useCallback, useEffect, useState } from 'react'

import * as movieApi from 'lib/movieApi'
import { errorMessage } from 'lib/errors'
import { Movie, MovieInput } from 'types/movie'

export default function useMovies() {
  const [movies, setMovies] = useState<Movie[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const reload = useCallback(async () => {
    setIsLoading(true)
    try {
      setMovies(await movieApi.fetchMovies())
      setError(null)
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    reload()
  }, [reload])

  // Runs a mutation, then refreshes the list from the server.
  // Re-throws so the caller can keep the form open on failure.
  const mutate = useCallback(
    async (action: () => Promise<unknown>) => {
      try {
        await action()
        setError(null)
      } catch (e) {
        setError(errorMessage(e))
        throw e
      }
      await reload()
    },
    [reload],
  )

  const create = (input: MovieInput) => mutate(() => movieApi.createMovie(input))
  const update = (id: string, input: MovieInput) => mutate(() => movieApi.updateMovie(id, input))
  const remove = (id: string) => mutate(() => movieApi.deleteMovie(id))

  return { movies, isLoading, error, create, update, remove }
}
