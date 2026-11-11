import Message from 'components/atoms/Message'
import MovieTableRow from 'components/molecules/MovieTableRow'
import { Movie } from 'types/movie'

import styles from 'styles/MovieTable.module.css'

type Props = {
  movies: Movie[]
  editingId: string | null
  onEdit: (movie: Movie) => void
  onDelete: (movie: Movie) => void
}

export default function MovieTable(props: Props) {
  const { movies, editingId, onEdit, onDelete } = props

  if (movies.length === 0) {
    return <Message>No movies yet. Add one above.</Message>
  }

  return (
    <table className={styles.table}>
      <thead>
        <tr>
          <th>ID</th>
          <th>Title</th>
          <th>ISBN</th>
          <th>Director</th>
          <th aria-label="Actions" />
        </tr>
      </thead>
      <tbody>
        {movies.map((movie) => (
          <MovieTableRow
            key={movie.id}
            movie={movie}
            isEditing={movie.id === editingId}
            onEdit={onEdit}
            onDelete={onDelete}
          />
        ))}
      </tbody>
    </table>
  )
}
