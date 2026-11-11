import { FormEvent, useState } from 'react'

import Button from 'components/atoms/Button'
import FormField from 'components/molecules/FormField'
import { Movie, MovieInput } from 'types/movie'

import styles from 'styles/MovieForm.module.css'

type Props = {
  // When set, the form edits this movie; otherwise it creates a new one
  movie?: Movie
  onSubmit: (input: MovieInput) => Promise<void>
  onCancel?: () => void
}

export default function MovieForm(props: Props) {
  const { movie, onSubmit, onCancel } = props

  const [title, setTitle] = useState(movie?.title ?? '')
  const [isbn, setIsbn] = useState(movie?.isbn ?? '')
  const [firstname, setFirstname] = useState(movie?.director?.firstname ?? '')
  const [lastname, setLastname] = useState(movie?.director?.lastname ?? '')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const isValid = title.trim() !== '' && isbn.trim() !== ''

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (!isValid) return

    setIsSubmitting(true)
    try {
      await onSubmit({
        title: title.trim(),
        isbn: isbn.trim(),
        director: { firstname: firstname.trim(), lastname: lastname.trim() },
      })
      if (!movie) {
        setTitle('')
        setIsbn('')
        setFirstname('')
        setLastname('')
      }
    } catch {
      // Error is surfaced by useMovies; keep the input so the user can retry
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <form className={styles.form} onSubmit={handleSubmit}>
      <h2>{movie ? `Edit movie #${movie.id}` : 'Add a movie'}</h2>
      <div className={styles.fields}>
        <FormField label="Title" value={title} onChange={(e) => setTitle(e.target.value)} required />
        <FormField label="ISBN" value={isbn} onChange={(e) => setIsbn(e.target.value)} required />
        <FormField
          label="Director first name"
          value={firstname}
          onChange={(e) => setFirstname(e.target.value)}
        />
        <FormField
          label="Director last name"
          value={lastname}
          onChange={(e) => setLastname(e.target.value)}
        />
      </div>
      <div className={styles.actions}>
        <Button type="submit" variant="primary" disabled={!isValid || isSubmitting}>
          {isSubmitting ? 'Saving…' : movie ? 'Save changes' : 'Add movie'}
        </Button>
        {onCancel && (
          <Button onClick={onCancel} disabled={isSubmitting}>
            Cancel
          </Button>
        )}
      </div>
    </form>
  )
}
